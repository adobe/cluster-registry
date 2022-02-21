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
	"strconv"
	"testing"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/database"
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

func (m mockDatabase) ListClusters(offset int, limit int, region string, environment string, status string) ([]registryv1.Cluster, int, bool, error) {
	return m.clusters, len(m.clusters), false, nil
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

func (m *mockDatabase) Status() error {
	return nil
}

type mockSQS struct {
	sqsiface.SQSAPI
	messages []*sqs.Message
}

func (m *mockSQS) GetQueueUrl(in *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {

	switch {
	case *in.QueueName == "dummy-que-name":
		queryUrl := "https://dummy-que-name.com"
		return &sqs.GetQueueUrlOutput{
			QueueUrl: &queryUrl,
		}, nil
	}

	return &sqs.GetQueueUrlOutput{}, errors.New("No sqs found with the name " + *in.QueueName)
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

	t.Log("Test initializing the sqs.")

	appConfig := &config.AppConfig{
		SqsEndpoint:  "dummy-url",
		SqsAwsRegion: "dummy-region",
	}

	s := NewSQS(appConfig)
	test.NotNil(s)
}
