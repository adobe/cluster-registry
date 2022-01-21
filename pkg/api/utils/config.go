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
	"log"
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

func LoadApiConfig() *AppConfig {
	awsRegion := getEnv("AWS_REGION", "", true)
	dbAwsRegion := getEnv("DB_AWS_REGION", "", true)
	dbEndpoint := getEnv("DB_ENDPOINT", "", true)
	dbTableName := getEnv("DB_TABLE_NAME", "", true)
	dbIndexName := getEnv("DB_INDEX_NAME", "", false)
	sqsEndpoint := getEnv("SQS_ENDPOINT", "", true)
	sqsAwsRegion := getEnv("SQS_AWS_REGION", "", true)
	sqsQueueName := getEnv("SQS_QUEUE_NAME", "", true)
	oidcClientId := getEnv("OIDC_CLIENT_ID", "", true)
	oidcIssuerUrl := getEnv("OIDC_ISSUER_URL", "", true)

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
	}
}

func LoadClientConfig() *AppConfig {
	sqsEndpoint := getEnv("SQS_ENDPOINT", "", true)
	sqsAwsRegion := getEnv("SQS_AWS_REGION", "", true)
	sqsQueueName := getEnv("SQS_QUEUE_NAME", "", true)

	return &AppConfig{
		SqsEndpoint:  sqsEndpoint,
		SqsAwsRegion: sqsAwsRegion,
		SqsQueueName: sqsQueueName,
	}
}

func getEnv(varName string, defaultValue string, required bool) string {

	varValue := os.Getenv(varName)
	if len(varValue) == 0 {
		if required == true {
			log.Fatalf("Environment variable '%s' not set. The app cannot start.", varName)
		} else {
			return defaultValue
		}
	}
	return varValue
}
