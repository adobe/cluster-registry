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
	"encoding/json"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/labstack/gommon/log"

	"github.com/adobe/cluster-registry/pkg/api/database"
	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	"github.com/adobe/cluster-registry/pkg/api/utils"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
)

const (
	egressTarget = "sqs"
)

// Consumer interface
type Consumer interface {
	Consume()
	worker(int)
	processMessage(*sqs.Message) error
	delete(m *sqs.Message) error
}

// consumer struct
type consumer struct {
	sqs             sqsiface.SQSAPI
	db              database.Db
	queueURL        string
	workerPool      int
	maxMessages     int64
	pollWaitSeconds int64
	retrySeconds    int
	met             monitoring.MetricsI
}

// NewConsumer - creates new message queue consumer
func NewConsumer(appConfig *utils.AppConfig, d database.Db, m monitoring.MetricsI) Consumer {
	sqsSvc := NewSQS(appConfig)

	urlResult, err := sqsSvc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &appConfig.SqsQueueName,
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	return &consumer{
		sqs:             sqsSvc,
		db:              d,
		queueURL:        *urlResult.QueueUrl,
		workerPool:      10,
		maxMessages:     1,
		pollWaitSeconds: 1,
		retrySeconds:    5,
		met:             m,
	}
}

// Consume - long pooling
func (c *consumer) Consume() {
	for w := 1; w <= c.workerPool; w++ {
		go c.worker(w)
	}
}

func (c *consumer) worker(id int) {
	for {
		start := time.Now()
		output, err := c.sqs.ReceiveMessage((&sqs.ReceiveMessageInput{
			QueueUrl: &c.queueURL,
			AttributeNames: aws.StringSlice([]string{
				"ClusterName", "SentTimestamp",
			}),
			MaxNumberOfMessages: aws.Int64(c.maxMessages),
			WaitTimeSeconds:     aws.Int64(c.pollWaitSeconds),
		}))
		elapsed := float64(time.Since(start)) / float64(time.Second)

		c.met.RecordEgressRequestCnt(egressTarget)
		c.met.RecordEgressRequestDur(egressTarget, elapsed)

		if err != nil {
			log.Error(err.Error())
			log.Info("Retrying in", c.retrySeconds, " seconds")
			time.Sleep(time.Duration(c.retrySeconds) * time.Second)
			continue
		}

		for _, m := range output.Messages {
			if err := c.processMessage(m); err != nil {
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

func (c *consumer) processMessage(m *sqs.Message) error {
	var rcvCluster registryv1.Cluster

	err := json.Unmarshal([]byte(*m.Body), &rcvCluster)
	if err != nil {
		log.Error("Failed to unmarshal message.")
		return err
	}

	clusterName := rcvCluster.Spec.Name

	msgTimestamp, err := strconv.ParseInt(*m.Attributes["SentTimestamp"], 10, 64)
	if err != nil {
		log.Error("Wrong time format for sqs message:", m.MessageId)
		return err
	}
	lastUpdated := time.Unix(0, msgTimestamp*int64(time.Millisecond))

	cluster, err := c.db.GetCluster(clusterName)
	if err != nil {
		log.Error("Failed to get cluster ", clusterName, " from database.")
		return err
	}

	if cluster == nil {
		rcvCluster.Spec.LastUpdated = lastUpdated.UTC().Format(time.RFC3339Nano)
		err = c.db.PutCluster(&rcvCluster)
		if err != nil {
			log.Error("Cluster ", clusterName, " failed to be created.")
			return err
		}
		log.Info("Cluster ", clusterName, " was created.")
		return nil
	}

	clusterTime, err := time.Parse(time.RFC3339Nano, cluster.Spec.LastUpdated)
	if err != nil {
		log.Warn("Wrong time format in database for: ", clusterName)
	} else if lastUpdated.Before(clusterTime) {
		log.Info("Cluster lastUpdated timestamp is too old. This update will be skip for ", clusterName)
		return nil
	}

	rcvCluster.Spec.LastUpdated = lastUpdated.UTC().Format(time.RFC3339Nano)
	err = c.db.PutCluster(&rcvCluster)
	if err != nil {
		log.Error("Cluster ", clusterName, " failed to be updated.")
		return err
	}

	log.Info("Cluster ", clusterName, " was updated.")
	return err
}

func (c *consumer) delete(m *sqs.Message) error {
	start := time.Now()
	_, err := c.sqs.DeleteMessage(
		&sqs.DeleteMessageInput{QueueUrl: &c.queueURL, ReceiptHandle: m.ReceiptHandle})
	elapsed := float64(time.Since(start)) / float64(time.Second)

	c.met.RecordEgressRequestCnt(egressTarget)
	c.met.RecordEgressRequestDur(egressTarget, elapsed)

	if err != nil {
		log.Error(err.Error())
	}
	return err
}
