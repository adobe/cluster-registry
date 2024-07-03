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
	"encoding/json"
	"fmt"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/aws/aws-sdk-go/aws"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SQS suite", func() {

	var (
		q           *Config
		appConfig   *config.AppConfig
		messageBody = map[string]string{
			"foo": "bar",
		}
	)

	BeforeEach(func() {
		var err error

		appConfig = &config.AppConfig{
			SqsEndpoint:  container.Endpoint,
			SqsAwsRegion: "dummy-region",
			SqsQueueName: "cluster-registry-local",
		}

		q, err = NewSQS(Config{
			AWSRegion:         appConfig.SqsAwsRegion,
			Endpoint:          appConfig.SqsEndpoint,
			QueueName:         appConfig.SqsQueueName,
			QueueURL:          fmt.Sprintf("%s/%s/%s", container.Endpoint, "1234567890", appConfig.SqsQueueName),
			BatchSize:         1,
			VisibilityTimeout: 120,
			WaitSeconds:       10,
			RunInterval:       5,
			RunOnce:           true,
			MaxHandlers:       10,
			BusyTimeout:       30,
		})
		Expect(err).To(BeNil())
	})

	AfterEach(func() {

	})

	Context("SQS tests", func() {

		It("should handle SQS status successfully", func() {
			err := q.Status()
			Expect(err).To(BeNil())
		})

		It("should successfully enqueue a message", func() {
			data, err := json.Marshal(messageBody)
			Expect(err).To(BeNil())

			err = q.Enqueue(context.Background(), []*awssqs.SendMessageBatchRequestEntry{
				{
					Id: aws.String("test-message"),
					MessageAttributes: map[string]*awssqs.MessageAttributeValue{
						MessageAttributeType: {
							DataType:    aws.String("String"),
							StringValue: aws.String(ClusterUpdateEvent),
						},
					},
					MessageBody: aws.String(string(data)),
				},
			})
			Expect(err).To(BeNil())
		})

		It("should successfully consume an enqueued message", func() {
			var count = 0
			q.RegisterHandler(func(msg *awssqs.Message) {
				defer GinkgoRecover()

				if msg == nil {
					return
				}

				Expect(*msg.Body).To(Equal(`{"foo":"bar"}`))
				count++

				err := q.Delete(msg)
				Expect(err).To(BeNil())
			})

			q.Poll()
			Eventually(count).Should(Equal(1))
		})
	})
})
