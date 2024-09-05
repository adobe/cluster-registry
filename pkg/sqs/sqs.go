/*
Copyright 2024 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package sqs

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sync"
	"time"
)

var logger logr.Logger

func init() {
	logger = ctrl.Log.WithName("sqs")
	ctrl.SetLogger(zap.New())
}

type Config struct {
	AWSRegion string
	QueueName string
	Endpoint  string
	QueueURL  string

	// Maximum number of time to attempt AWS service connection
	MaxRetries int

	// Maximum number of messages to retrieve per batch
	BatchSize int64

	// The maximum poll time (0 <= 20)
	WaitSeconds int64

	// Once a message is received by a consumer, the maximum time in seconds till others can see this
	VisibilityTimeout int64

	// Poll only once and exit
	RunOnce bool

	// Poll every X seconds defined by this value
	RunInterval int

	// Maximum number of handlers to spawn for batch processing
	MaxHandlers int

	// BusyTimeout in seconds
	BusyTimeout int

	svc          *sqs.SQS
	handlerCount int
	mutex        *sync.Mutex
	handler      func(msg *sqs.Message)
}

type SQS interface {
	Poll()
	Delete(msg *sqs.Message) error
	Enqueue(msgBatch []*sqs.SendMessageBatchRequestEntry) error
	RegisterHandler(handler func(msg *sqs.Message))
	ChangeVisibilityTimeout(msg *sqs.Message, seconds int64) bool
	Status() error
}

// NewSQS - create new SQS instance
func NewSQS(cfg Config) (*Config, error) {
	// Validate parameters
	validateErr := validateConfig(cfg)
	if validateErr != nil {
		logger.Error(validateErr, "invalid SQS Config")
		return nil, validateErr
	}

	// Create AWS Config
	awsConfig := aws.NewConfig().
		WithRegion(cfg.AWSRegion).
		WithMaxRetries(cfg.MaxRetries).
		WithEndpoint(cfg.Endpoint)
	if awsConfig == nil {
		logger.Info("Invalid AWS Config")
		return nil, errors.New("something is wrong with your AWS config parameters")
	}

	// Establish a session
	newSession := session.Must(session.NewSession(awsConfig))
	if newSession == nil {
		logger.Info("Unable to create session")
		return nil, errors.New("unable to create session")
	}

	// Create a service connection
	svc := sqs.New(newSession)
	if svc == nil {
		logger.Info("Unable to connect to SQS")
		return nil, errors.New("unable to create a service connection with AWS SQS")
	}

	if cfg.QueueURL == "" {
		logger.Info("Fetching queue URL")
		res, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
			QueueName: &cfg.QueueName,
		})
		if err != nil {
			logger.Info("Unable to fetch queue name", err)
			return nil, errors.New("unable to get queue name")
		}

		cfg.QueueURL = *res.QueueUrl
	}
	logger.Info("Connected to Queue")

	cfg.svc = svc
	cfg.mutex = &sync.Mutex{}
	return &cfg, nil
}

// Poll for messages in the queue
func (s *Config) Poll() {
	if s.svc == nil {
		logger.Error(nil, "No service connection")
		return
	}

	wg := sync.WaitGroup{}
	batch := 0

	for {
		batch++
		childLogger := ctrl.Log.WithName(fmt.Sprintf("sqs-batch-%d", batch))
		childLogger.Info("Start receiving messages")

		maxMsgs := s.BatchSize

		// Is running at capacity?
		if s.MaxHandlers > 0 {
			for s.handlerCount >= s.MaxHandlers {
				childLogger.Info("Reached max handler count")
				childLogger.Info("Going to wait state", "timeout", s.BusyTimeout)
				<-time.After(time.Duration(s.BusyTimeout) * time.Second)
			}
			availableHandlers := int64(s.MaxHandlers - s.handlerCount)
			if availableHandlers < maxMsgs {
				maxMsgs = availableHandlers
			}
		}

		childLogger.Info("Polling for messages", "maxMessages", maxMsgs)

		result, err := s.svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:                    &s.QueueURL,
			MaxNumberOfMessages:         &maxMsgs,
			WaitTimeSeconds:             &s.WaitSeconds,
			VisibilityTimeout:           &s.VisibilityTimeout,
			MessageAttributeNames:       aws.StringSlice([]string{"All"}),
			MessageSystemAttributeNames: aws.StringSlice([]string{"All"}),
			AttributeNames:              aws.StringSlice([]string{"All"}),
		})

		// Retrieve error?
		if err != nil {
			childLogger.Info("ReceiveMessageError:", err)
			break
		}

		// Message log
		if len(result.Messages) == 0 {
			childLogger.Info("Queue is empty")
		} else {
			childLogger.Info("Fetched messages", "count", len(result.Messages))
		}

		// Process messages
		for _, msg := range result.Messages {
			if s.handler == nil {
				childLogger.Info("No handler registered")
			} else {
				s.handlerCount++
				wg.Add(1)

				go func(m *sqs.Message) {
					s.handler(m)

					s.mutex.Lock()
					s.handlerCount--
					s.mutex.Unlock()

					wg.Done()
				}(msg)
			}

			childLogger.Info("Spawned handler", "messageId", *msg.MessageId)
		}

		if s.RunOnce {
			childLogger.Info(`Exiting since configured to run once`)
			break
		} else {
			childLogger.Info("Waiting before polling for next batch", "interval", s.RunInterval)
			<-time.After(time.Duration(s.RunInterval) * time.Second)
		}

		childLogger.Info("Finished polling")
	}

	wg.Wait()
}

// Enqueue messages to SQS
func (s *Config) Enqueue(ctx context.Context, msgBatch []*sqs.SendMessageBatchRequestEntry) error {
	if s.svc == nil {
		return errors.New("no service connection")
	}

	logger.Info("Enqueuing messages", "count", len(msgBatch))

	result, err := s.svc.SendMessageBatchWithContext(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: &s.QueueURL,
		Entries:  msgBatch,
	})

	if err != nil {
		logger.Info("Enqueue error:", err)
		return err
	}

	for _, failed := range result.Failed {
		logger.Info("Failed to enqueue message", "messageId", *failed.Id)
	}
	for _, success := range result.Successful {
		logger.Info("Enqueued message", "messageId", *success.MessageId)
	}

	return err
}

// Delete a SQS message from the queue
func (s *Config) Delete(msg *sqs.Message) error {
	logger.Info("Deleting message", "messageId", *msg.MessageId)

	_, err := s.svc.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &s.QueueURL,
		ReceiptHandle: msg.ReceiptHandle,
	})

	return err
}

// RegisterHandler : A method to register a custom Poll Handling method
func (s *Config) RegisterHandler(handler func(msg *sqs.Message)) {
	s.handler = handler
}

// ChangeVisibilityTimeout : Method to change visibility timeout of a message.
func (s *Config) ChangeVisibilityTimeout(msg *sqs.Message, seconds int64) bool {
	logger.Info("changing message visibility timeout", "messageId", *msg.MessageId)

	if s.svc == nil {
		logger.Error(nil, "SQS Connection failed")
		return false
	}

	strURL := &s.QueueURL
	receiptHandle := *msg.ReceiptHandle

	changeMessageVisibilityInput := sqs.ChangeMessageVisibilityInput{}

	changeMessageVisibilityInput.SetQueueUrl(*strURL)
	changeMessageVisibilityInput.SetReceiptHandle(receiptHandle)
	changeMessageVisibilityInput.SetVisibilityTimeout(seconds)

	_, err := s.svc.ChangeMessageVisibility(&changeMessageVisibilityInput)

	if err == nil {
		logger.Info("successfully changed message visibility timeout", "messageId", *msg.MessageId)
		return true
	} else {
		logger.Info("failed to change message visibility timeout", "messageId", *msg.MessageId)
	}

	return false
}

func (s *Config) Status() error {
	_, err := s.svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &s.QueueName,
	})

	// TODO: add metrics integration

	return err
}

func validateConfig(cfg Config) error {
	if cfg.AWSRegion == "" {
		return errors.New("AWSRegion is required")
	}

	if cfg.Endpoint == "" && cfg.QueueURL == "" {
		return errors.New("A valid SQS Endpoint is required")
	}

	if cfg.QueueName == "" && cfg.QueueURL == "" {
		return errors.New("A valid SQS Queue Name is required")
	}

	if cfg.QueueURL == "" && cfg.Endpoint == "" && cfg.QueueName == "" {
		return errors.New("A valid SQS Queue URL is required")
	}

	if cfg.BatchSize < 0 || cfg.BatchSize > 10 {
		return errors.New("BatchSize should be between 1-10")
	}

	if cfg.WaitSeconds < 0 || cfg.WaitSeconds > 20 {
		return errors.New("WaitSeconds should be between 1-20")
	}

	if cfg.VisibilityTimeout < 0 || cfg.VisibilityTimeout > 12*60*60 {
		return errors.New("VisibilityTimeout should be between 1-43200")
	}

	return nil
}
