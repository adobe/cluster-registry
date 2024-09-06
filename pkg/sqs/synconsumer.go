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
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/labstack/gommon/log"

	"github.com/adobe/cluster-registry/pkg/config"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
)

// consumer struct
type synconsumer struct {
	sqs             sqsiface.SQSAPI
	queueURL        string
	workerPool      int
	maxMessages     int64
	pollWaitSeconds int64
	retrySeconds    int
	messageHandler  func(*sqs.Message) error
}

// NewSyncConsumer - creates a new SQS message queue consumer
// used by the sync consumer service
// TODO: add metrics later
func NewSyncConsumer(sqsSvc sqsiface.SQSAPI, appConfig *config.AppConfig, h func(*sqs.Message) error) Consumer {

	urlResult, err := sqsSvc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &appConfig.SqsQueueName,
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	return &synconsumer{
		sqs:             sqsSvc,
		queueURL:        *urlResult.QueueUrl,
		workerPool:      10,
		maxMessages:     1,
		pollWaitSeconds: 1,
		retrySeconds:    5,
		messageHandler:  h,
	}
}

// Status verifies the status/connectivity of the sqs service
func (c *synconsumer) Status(appConfig *config.AppConfig, m monitoring.MetricsI) error {
	_, err := c.sqs.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &appConfig.SqsQueueName,
	})

	if err != nil {
		log.Error(err.Error())
	}

	return err
}

// Consume - long pooling
func (c *synconsumer) Consume() {
	var wg sync.WaitGroup

	for w := 1; w <= c.workerPool; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			c.worker(w)
		}(w)
	}
	wg.Wait()
}

func (c *synconsumer) worker(id int) {
	for {
		output, err := c.sqs.ReceiveMessage((&sqs.ReceiveMessageInput{
			QueueUrl: &c.queueURL,
			AttributeNames: aws.StringSlice([]string{
				"ClusterName", "SentTimestamp",
			}),
			MaxNumberOfMessages: aws.Int64(c.maxMessages),
			WaitTimeSeconds:     aws.Int64(c.pollWaitSeconds),
		}))

		if err != nil {
			log.Error(err.Error())
			log.Info("Retrying in", c.retrySeconds, " seconds")
			time.Sleep(time.Duration(c.retrySeconds) * time.Second)
			continue
		}

		for _, m := range output.Messages {
			log.Debug("Messsage ID: ", *m.MessageId)
			log.Debug("Message Body: ", *m.Body)

			err := c.processMessage(m)

			if err != nil {
				log.Error(err.Error())
				continue
			}
			err = c.delete(m)
			if err != nil {
				log.Error(err.Error())
			}
		}
	}
}

// processMessage - process the recieved message
func (c *synconsumer) processMessage(m *sqs.Message) error {

	err := c.messageHandler(m)
	return err
}

func (c *synconsumer) delete(m *sqs.Message) error {

	_, err := c.sqs.DeleteMessage(
		&sqs.DeleteMessageInput{QueueUrl: &c.queueURL, ReceiptHandle: m.ReceiptHandle})

	if err != nil {
		log.Error(err.Error())
	}
	return err
}
