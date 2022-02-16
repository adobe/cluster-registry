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
	"fmt"
	"testing"

	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/api/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
)

func TestDeleteMessage(t *testing.T) {
	test := assert.New(t)

	t.Log("Test deleting messages from sqs queue.")

	tcs := []struct {
		name          string
		sqsMessages   []*sqs.Message
		expectedError error
	}{
		{
			name: "delete message",
			sqsMessages: []*sqs.Message{
				{Body: aws.String(`{"apiVersion":"registry.ethos.adobe.com/v1","kind":"Cluster","metadata":{"name":"cluster1","namespace":"cluster-registry"},"spec":{"name":"cluster1","registeredAt":"2019-02-13T06:15:32Z","lastUpdated":"2020-02-13T06:15:32Z","status":"Deprecated","phase":"Running","tags":{"onboarding":"off","scaling":"off"}}}`)},
				{Body: aws.String(`{"apiVersion":"registry.ethos.adobe.com/v1","kind":"Cluster","metadata":{"name":"cluster2","namespace":"cluster-registry"},"spec":{"name":"cluster2","registeredAt":"2019-02-13T06:15:32Z","lastUpdated":"2020-02-13T06:15:32Z","status":"Deprecated","phase":"Running","tags":{"onboarding":"off","scaling":"off"}}}`)},
			},
			expectedError: nil,
		}}

	for _, tc := range tcs {
		m := monitoring.NewMetrics("cluster_registry_api_sqs_test", true)
		c := &consumer{
			sqs:             &mockSQS{messages: tc.sqsMessages},
			db:              &mockDatabase{clusters: nil},
			queueURL:        "mock-queue",
			workerPool:      1,
			maxMessages:     1,
			pollWaitSeconds: 1,
			retrySeconds:    5,
			metrics:         m,
		}

		t.Logf("\tTest %s:\tWhen deleting a message from sqs queue.", tc.name)

		err := c.delete(tc.sqsMessages[0])
		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		}

		queueDepth, _ := getQueueDepth(c.sqs)
		test.Equal(queueDepth, 1)

		err = c.delete(tc.sqsMessages[0])
		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		}

		queueDepth, _ = getQueueDepth(c.sqs)
		test.Equal(queueDepth, 0)
	}
}

func TestStatusHealthCheck(t *testing.T) {
	test := assert.New(t)

	t.Log("Test checking the health of the sqs")

	tcs := []struct {
		name          string
		sqsQueueName  string
		queueURL      string
		expectedError error
	}{
		{
			name:          "successful health check",
			sqsQueueName:  "dummy-que-name",
			queueURL:      "https://dummy-que-name.com",
			expectedError: nil,
		},
		{
			name:          "unsuccessful health check",
			sqsQueueName:  "missing-dummy-que-name",
			queueURL:      "",
			expectedError: fmt.Errorf("No sqs found with the name missing-dummy-que-name"),
		},
	}

	var err error
	for _, tc := range tcs {
		appConfig := &utils.AppConfig{
			SqsQueueName: tc.sqsQueueName,
		}
		m := monitoring.NewMetrics("err_count_sqs_test", true)
		c := &consumer{
			sqs:     &mockSQS{},
			db:      &mockDatabase{clusters: nil},
			metrics: m,
		}

		t.Logf("\tTest %s", tc.name)

		err = c.Status(appConfig, m)
		test.Equal(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError))

	}
}

func TestProcessMessage(t *testing.T) {
	test := assert.New(t)

	t.Log("Test processing messages from sqs queue.")

	tcs := []struct {
		name             string
		dbClusters       []registryv1.Cluster
		sqsMessages      []*sqs.Message
		expectedClusters []registryv1.Cluster
		expectedError    error
	}{
		{
			name: "new cluster",
			dbClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				},
			},
			},
			sqsMessages: []*sqs.Message{
				{
					Body: aws.String(`{"spec":{"name":"cluster2","registeredAt":"2019-02-13T06:15:32Z","status":"Active","phase":"Running","tags":{"onboarding":"on","scaling":"off"}}}`),
					Attributes: map[string]*string{
						"SentTimestamp": aws.String("1627321893000"),
					},
				},
			},
			expectedClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				}},
				{
					Spec: registryv1.ClusterSpec{
						Name:         "cluster2",
						LastUpdated:  "2021-07-26T17:51:33Z",
						RegisteredAt: "2019-02-13T06:15:32Z",
						Status:       "Active",
						Phase:        "Running",
						Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
					}}},
			expectedError: nil,
		},
		{
			name: "older update timestamp",
			dbClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				},
			},
			},
			sqsMessages: []*sqs.Message{
				{
					Body: aws.String(`{"apiVersion":"registry.ethos.adobe.com/v1","kind":"Cluster","metadata":{"name":"cluster1","namespace":"cluster-registry"},"spec":{"name":"cluster1","registeredAt":"2019-02-13T06:15:32Z","status":"Deprecated","phase":"Running","tags":{"onboarding":"off","scaling":"off"}}}`),
					Attributes: map[string]*string{
						"SentTimestamp": aws.String("1581532782000"),
					},
				},
			},
			expectedClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				}},
			},
			expectedError: nil,
		},
		{
			name: "wrong sqs message",
			dbClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				},
			},
			},
			sqsMessages: []*sqs.Message{
				{Body: aws.String(`{this is wrong}`)},
			},
			expectedClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				}},
			},
			expectedError: fmt.Errorf("invalid character 't' looking for beginning of object key string"),
		},
		{
			name: "wrong timestamp format",
			dbClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				},
			},
			},
			sqsMessages: []*sqs.Message{
				{
					Body: aws.String(`{"spec":{"name":"cluster2","registeredAt":"2019-02-13T06:15:32Z","status":"Active","phase":"Running","tags":{"onboarding":"on","scaling":"off"}}}`),
					Attributes: map[string]*string{
						"SentTimestamp": aws.String("1234abc"),
					},
				},
			},
			expectedClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				}},
			},
			expectedError: fmt.Errorf("strconv.ParseInt: parsing \"1234abc\": invalid syntax"),
		},
	}

	for _, tc := range tcs {
		c := &consumer{
			sqs:             &mockSQS{messages: tc.sqsMessages},
			db:              &mockDatabase{clusters: tc.dbClusters},
			queueURL:        "mock-queue",
			workerPool:      1,
			maxMessages:     1,
			pollWaitSeconds: 1,
			retrySeconds:    5,
			metrics:         monitoring.NewMetrics("cluster_registry_api_sqs_test", true),
		}

		t.Logf("\tTest %s:\tWhen processing a message from sqs queue.", tc.name)

		err := c.processMessage(tc.sqsMessages[0])
		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.NoError(err)
		}

		listClusters, _, _, err := c.db.ListClusters(0, 10, "", "", "")
		if err != nil {
			test.Error(err, "cannot list cluster from database")
		}

		test.Equal(tc.expectedClusters, listClusters)
	}
}
