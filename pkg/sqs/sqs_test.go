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
	"github.com/adobe/cluster-registry/pkg/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SQS suite", func() {

	var q *Config
	var appConfig *config.AppConfig

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
			BatchSize:         10,
			VisibilityTimeout: 120,
			WaitSeconds:       5,
			RunInterval:       20,
			RunOnce:           false,
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

		// TODO: add more tests
	})
})
