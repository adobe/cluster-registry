/*
Copyright 2021 Adobe. All rights reserved.
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
	"encoding/json"
	"time"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/client"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/labstack/gommon/log"
)

// Producer interface
type Producer interface {
	Send(context.Context, *registryv1.Cluster) error
}

// producer struct
type producer struct {
	sqs      sqsiface.SQSAPI
	queueURL string
	metrics  monitoring.MetricsI
}

// NewProducer - create new message queue producer
func NewProducer(appConfig *config.AppConfig, m monitoring.MetricsI) Producer {

	sqsSvc := NewSQS(appConfig)

	urlResult, err := sqsSvc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &appConfig.SqsQueueName,
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	return &producer{
		sqs:      sqsSvc,
		queueURL: *urlResult.QueueUrl,
		metrics:  m,
	}
}

// Send message in sqs queue
func (p *producer) Send(ctx context.Context, c *registryv1.Cluster) error {

	o, err := json.Marshal(c)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	start := time.Now()
	result, err := p.sqs.SendMessageWithContext(ctx, &sqs.SendMessageInput{
		DelaySeconds: aws.Int64(10),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"ClusterName": {
				DataType:    aws.String("String"),
				StringValue: aws.String(c.Spec.Name),
			},
		},
		MessageBody: aws.String(string(o)),
		QueueUrl:    &p.queueURL,
	})
	elapsed := float64(time.Since(start)) / float64(time.Second)

	p.metrics.RecordEgressRequestCnt(egressTarget)
	p.metrics.RecordEgressRequestDur(egressTarget, elapsed)

	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("Message ", *result.MessageId, " sent successfully.")
	return nil
}
