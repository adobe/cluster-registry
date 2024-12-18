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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
	"log"
	"os"
	"time"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/client"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"gopkg.in/yaml.v2"
)

func main() {
	var clusters []registryv1.Cluster

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	m := monitoring.NewMetrics()
	m.Init(false)
	appConfig, err := config.LoadApiConfig()
	if err != nil {
		log.Fatalf("Cannot load the api configuration: '%v'", err.Error())
	}

	q, err := sqs.NewSQS(sqs.Config{
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
	if err != nil {
		log.Panicf("Error while trying to create SQS client: %v", err.Error())
	}

	input_file := flag.String("input-file", "../db/dummy-data.yaml", "yaml file path")
	flag.Parse()

	data, err := os.ReadFile(*input_file)
	if err != nil {
		log.Panicf("Error while trying to read file: %v", err.Error())
	}

	err = yaml.Unmarshal(data, &clusters)
	if err != nil {
		log.Panicf("Error while trying to unmarshal data: %v", err.Error())
	}

	id, err := uuid.NewUUID()
	if err != nil {
		log.Panic(err.Error())
	}

	for _, cluster := range clusters {
		data, _ := json.Marshal(cluster)
		err = q.Enqueue(ctx, []*awssqs.SendMessageBatchRequestEntry{
			{
				Id:           aws.String(id.String()),
				DelaySeconds: aws.Int64(10),
				MessageAttributes: map[string]*awssqs.MessageAttributeValue{
					"Type": {
						DataType:    aws.String("String"),
						StringValue: aws.String(sqs.ClusterUpdateEvent),
					},
					"ClusterName": {
						DataType:    aws.String("String"),
						StringValue: aws.String(cluster.Spec.Name),
					},
					"SkipCacheInvalidation": {
						DataType:    aws.String("Bool"),
						StringValue: aws.String(fmt.Sprintf("%t", false)),
					},
				},
				MessageBody: aws.String(string(data)),
			},
		})
		if err != nil {
			log.Panicf("Error sending message to sqs: %v", err.Error())
		}
	}

	fmt.Println("Data successfully added into the queue.")
}
