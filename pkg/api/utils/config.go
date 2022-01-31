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

package utils

import (
	"fmt"
	"os"
)

type AppConfig struct {
	AwsRegion     string
	DbEndpoint    string
	DbAwsRegion   string
	DbTableName   string
	DbIndexName   string
	SqsEndpoint   string
	SqsAwsRegion  string
	SqsQueueName  string
	OidcClientId  string
	OidcIssuerUrl string
}

func LoadApiConfig() (*AppConfig, error) {
	awsRegion := getEnv("AWS_REGION", "")
	if awsRegion == "" {
		return nil, fmt.Errorf("Environment variable AWS_REGION is not set.")
	}

	dbAwsRegion := getEnv("DB_AWS_REGION", "")
	if dbAwsRegion == "" {
		return nil, fmt.Errorf("Environment variable DB_AWS_REGION is not set.")
	}

	dbEndpoint := getEnv("DB_ENDPOINT", "")
	if dbEndpoint == "" {
		return nil, fmt.Errorf("Environment variable DB_ENDPOINT is not set.")
	}

	dbTableName := getEnv("DB_TABLE_NAME", "")
	if dbTableName == "" {
		return nil, fmt.Errorf("Environment variable DB_TABLE_NAME is not set.")
	}

	dbIndexName := getEnv("DB_INDEX_NAME", "")

	sqsEndpoint := getEnv("SQS_ENDPOINT", "")
	if sqsEndpoint == "" {
		return nil, fmt.Errorf("Environment variable SQS_ENDPOINT is not set.")
	}

	sqsAwsRegion := getEnv("SQS_AWS_REGION", "")
	if sqsAwsRegion == "" {
		return nil, fmt.Errorf("Environment variable SQS_AWS_REGION is not set.")
	}

	sqsQueueName := getEnv("SQS_QUEUE_NAME", "")
	if sqsQueueName == "" {
		return nil, fmt.Errorf("Environment variable SQS_QUEUE_NAME is not set.")
	}

	oidcClientId := getEnv("OIDC_CLIENT_ID", "")
	if oidcClientId == "" {
		return nil, fmt.Errorf("Environment variable OIDC_CLIENT_ID is not set.")
	}

	oidcIssuerUrl := getEnv("OIDC_ISSUER_URL", "")
	if oidcIssuerUrl == "" {
		return nil, fmt.Errorf("Environment variable OIDC_ISSUER_URL is not set.")
	}

	return &AppConfig{
		AwsRegion:     awsRegion,
		DbEndpoint:    dbEndpoint,
		DbAwsRegion:   dbAwsRegion,
		DbTableName:   dbTableName,
		DbIndexName:   dbIndexName,
		SqsEndpoint:   sqsEndpoint,
		SqsAwsRegion:  sqsAwsRegion,
		SqsQueueName:  sqsQueueName,
		OidcClientId:  oidcClientId,
		OidcIssuerUrl: oidcIssuerUrl,
	}, nil
}

func LoadClientConfig() (*AppConfig, error) {
	sqsEndpoint := getEnv("SQS_ENDPOINT", "")
	if sqsEndpoint == "" {
		return nil, fmt.Errorf("Environment variable SQS_ENDPOINT is not set.")
	}

	sqsAwsRegion := getEnv("SQS_AWS_REGION", "")
	if sqsAwsRegion == "" {
		return nil, fmt.Errorf("Environment variable SQS_AWS_REGION is not set.")
	}

	sqsQueueName := getEnv("SQS_QUEUE_NAME", "")
	if sqsQueueName == "" {
		return nil, fmt.Errorf("Environment variable SQS_QUEUE_NAME is not set.")
	}

	return &AppConfig{
		SqsEndpoint:  sqsEndpoint,
		SqsAwsRegion: sqsAwsRegion,
		SqsQueueName: sqsQueueName,
	}, nil
}

func getEnv(varName string, defaultValue string) string {
	varValue := os.Getenv(varName)
	if len(varValue) == 0 {
		return defaultValue
	}
	return varValue
}
