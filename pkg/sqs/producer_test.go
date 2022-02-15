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
	"log"
	"testing"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/client"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
)

func TestSendMessage(t *testing.T) {
	test := assert.New(t)

	t.Log("Test sending a messages to sqs queue.")

	tcs := []struct {
		name          string
		cluster       registryv1.Cluster
		expectedError error
	}{
		{
			name: "create cluster",
			cluster: registryv1.Cluster{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Deprecated",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "off", "scaling": "off"},
				}},
			expectedError: nil,
		}}

	for _, tc := range tcs {
		m := monitoring.NewMetrics()
		m.Init(true)
		p := producer{
			sqs:      &mockSQS{},
			queueURL: "mock-queue",
			metrics:  m,
		}

		t.Logf("\tTest %s:\tWhen sending a message to sqs queue.", tc.name)

		err := p.Send(context.TODO(), &tc.cluster)
		if err != nil {
			log.Panicf("Error sending message to sqs: %v", err.Error())
		}

		output, err := p.sqs.ReceiveMessage((&sqs.ReceiveMessageInput{
			QueueUrl:            &p.queueURL,
			MaxNumberOfMessages: aws.Int64(1),
			WaitTimeSeconds:     aws.Int64(1),
		}))
		test.NoError(err)

		var rcvCluster registryv1.Cluster
		err = json.Unmarshal([]byte(*output.Messages[0].Body), &rcvCluster)
		test.NoError(err)

		test.Equal(tc.cluster, rcvCluster)
	}
}
