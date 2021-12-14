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
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/adobe/cluster-registry/pkg/api/database"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/assert"
)

// mockDatabase database.db
type mockDatabase struct {
	database.Db
	clusters []registryv1.Cluster
}

func (m mockDatabase) GetCluster(name string) (*registryv1.Cluster, error) {
	for _, c := range m.clusters {
		if c.Spec.Name == name {
			return &c, nil
		}
	}
	return nil, nil
}

func (m mockDatabase) ListClusters(region string, environment string, businessUnit string, status string) ([]registryv1.Cluster, int, error) {
	return m.clusters, len(m.clusters), nil
}

func (m *mockDatabase) PutCluster(cluster *registryv1.Cluster) error {
	found := false

	for i, c := range m.clusters {
		if c.Spec.Name == cluster.Spec.Name {
			m.clusters[i] = *cluster
			found = true
		}
	}

	if !found {
		m.clusters = append(m.clusters, *cluster)
	}
	return nil
}

func (m *mockDatabase) DeleteCluster(name string) error {
	for i, c := range m.clusters {
		if c.Spec.Name == name {
			m.clusters = append(m.clusters[:i], m.clusters[i+1:]...)
		}
	}
	return nil
}

type mockSQS struct {
	sqsiface.SQSAPI
	messages []*sqs.Message
}

func (m *mockSQS) SendMessageWithContext(ctx aws.Context, in *sqs.SendMessageInput, r ...request.Option) (*sqs.SendMessageOutput, error) {
	m.messages = append(m.messages, &sqs.Message{
		Body: in.MessageBody,
	})
	messageID := "TWVzc2FnZUlkCg=="

	return &sqs.SendMessageOutput{
		MessageId: &messageID,
	}, nil
}

func (m *mockSQS) DeleteMessage(in *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	if len(m.messages) == 0 {
		return nil, errors.New("no messages to delete")
	}
	m.messages = m.messages[1:]
	return &sqs.DeleteMessageOutput{}, nil
}

func (m *mockSQS) ReceiveMessage(in *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	if len(m.messages) == 0 {
		return &sqs.ReceiveMessageOutput{}, nil
	}
	response := m.messages[0:1]
	m.messages = m.messages[1:]

	return &sqs.ReceiveMessageOutput{
		Messages: response,
	}, nil
}

// Used to get queue length - ApproximateNumberOfMessages
func (m *mockSQS) GetQueueAttributes(in *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {

	response := make(map[string]*string)
	value := strconv.Itoa(len(m.messages))

	response["locationNameKey"] = in.AttributeNames[0]
	response["locationNameValue"] = &value

	return &sqs.GetQueueAttributesOutput{
		Attributes: response,
	}, nil
}

func getQueueDepth(s sqsiface.SQSAPI) (int, error) {
	attributeName := "ApproximateNumberOfMessages"
	params := &sqs.GetQueueAttributesInput{
		QueueUrl: aws.String("mock-queue"),
		AttributeNames: []*string{
			&attributeName,
		},
	}

	r, err := s.GetQueueAttributes(params)
	if err != nil {
		log.Error(err.Error())
		return -1, err
	}

	result, _ := strconv.Atoi(*r.Attributes["locationNameValue"])
	return result, nil
}

func TestNewSqs(t *testing.T) {
	test := assert.New(t)

	os.Setenv("SQS_ENDPOINT", "dummy-url")
	os.Setenv("SQS_AWS_REGION", "dummy-region")

	s := NewSQS()
	test.NotNil(s)
}

func TestGetSqsAwsRegion(t *testing.T) {
	test := assert.New(t)
	tcs := []struct {
		name            string
		endpointURL     string
		envSqsAwsRegion string
		expectedRegion  string
		expectedError   error
	}{
		{
			name:            "get from endpoint",
			endpointURL:     "https://sqs.us-west-2.amazonaws.com/myaccountid/myqueue",
			envSqsAwsRegion: "",
			expectedRegion:  "us-west-2",
			expectedError:   nil,
		},
		{
			name:            "get from env var",
			endpointURL:     "https://sqs.amazonaws.com/myaccountid/myqueue",
			envSqsAwsRegion: "us-west-2",
			expectedRegion:  "us-west-2",
			expectedError:   nil,
		},
		{
			name:            "error",
			endpointURL:     "https://sqs.amazonaws.com/myaccountid/myqueue",
			envSqsAwsRegion: "",
			expectedRegion:  "",
			expectedError:   fmt.Errorf("cannot get sqs aws region from sqsEndpoint or from environment variables"),
		},
	}

	for _, tc := range tcs {
		os.Setenv("SQS_AWS_REGION", tc.envSqsAwsRegion)
		awsRegion, err := getSqsAwsRegion(tc.endpointURL)

		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.NoError(err)
		}
		test.Equal(tc.expectedRegion, awsRegion)
	}
}

func TestGetSqsQueueName(t *testing.T) {
	test := assert.New(t)
	tcs := []struct {
		name              string
		endpointURL       string
		envSqsQueueName   string
		expectedQueueName string
		expectedError     error
	}{
		{
			name:              "get from endpoint",
			endpointURL:       "https://sqs.us-west-2.amazonaws.com/myaccountid/myqueue",
			envSqsQueueName:   "",
			expectedQueueName: "myqueue",
			expectedError:     nil,
		},
		{
			name:              "get from env var",
			endpointURL:       "https://sqs.us-west-2.amazonaws.com",
			envSqsQueueName:   "myqueue",
			expectedQueueName: "myqueue",
			expectedError:     nil,
		},
		{
			name:              "error",
			endpointURL:       "https://sqs.us-west-2.amazonaws.com",
			envSqsQueueName:   "",
			expectedQueueName: "",
			expectedError:     fmt.Errorf("cannot get sqs queue name from sqsEndpoint or from environment variables"),
		},
	}

	for _, tc := range tcs {
		os.Setenv("SQS_QUEUE_NAME", tc.envSqsQueueName)
		queueName, err := getSqsQueueName(tc.endpointURL)

		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.NoError(err)
		}
		test.Equal(tc.expectedQueueName, queueName)
	}
}
