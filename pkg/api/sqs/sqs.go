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
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

var sqsEndpoint string

// NewSQS - create new SQS instance
func NewSQS() sqsiface.SQSAPI {

	sqsEndpoint = os.Getenv("SQS_ENDPOINT")
	if sqsEndpoint == "" {
		log.Fatal("SqS endpoint not set.")
	}

	awsRegion, err := getSqsAwsRegion(sqsEndpoint)
	if err != nil {
		log.Fatal(err.Error())
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:   aws.String(awsRegion),
			Endpoint: aws.String(sqsEndpoint),
		},
	}))

	return sqs.New(sess)
}

func getSqsAwsRegion(sqsEndpoint string) (string, error) {
	endpointParts := strings.Split(sqsEndpoint, ".")
	var awsRegion string
	if len(endpointParts) == 4 { // https://sqs.us-west-2.amazonaws.com/myaccountid/myqueue
		awsRegion = endpointParts[1]
	} else {
		awsRegion = os.Getenv("SQS_AWS_REGION")
	}
	if awsRegion == "" {
		return "", errors.New("cannot get sqs aws region from sqsEndpoint or from environment variables")
	}
	return awsRegion, nil
}

func getSqsQueueName(sqsEndpoint string) (string, error) {
	endpointParts := strings.Split(sqsEndpoint, "/")
	var sqsQueueName string
	if len(endpointParts) == 5 { // https://sqs.us-west-2.amazonaws.com/myaccountid/myqueue
		sqsQueueName = endpointParts[4]
	} else {
		sqsQueueName = os.Getenv("SQS_QUEUE_NAME")
	}
	if sqsQueueName == "" {
		return "", errors.New("cannot get sqs queue name from sqsEndpoint or from environment variables")
	}
	return sqsQueueName, nil
}
