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

package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	test := assert.New(t)

	t.Log("Test getting a single variable from environment.")

	tcs := []struct {
		name          string
		varName       string
		varValue      string
		defaultValue  string
		expectedValue string
	}{
		{
			name:          "required env var",
			varName:       "OIDC_ISSUER_URL",
			varValue:      "http://fake-oid-url",
			defaultValue:  "",
			expectedValue: "http://fake-oid-url",
		},
		{
			name:          "optional env var",
			varName:       "OIDC_ISSUER_URL",
			varValue:      "",
			defaultValue:  "http://fake-oid-url",
			expectedValue: "http://fake-oid-url",
		},
	}

	for _, tc := range tcs {
		_ = os.Setenv(tc.varName, tc.varValue)

		t.Logf("\tTest %s:\tWhen getting environment variable %s", tc.name, tc.varName)

		value := getEnv(tc.varName, tc.defaultValue)
		test.Equal(tc.expectedValue, value)
	}
}

func TestLoadApiConfig(t *testing.T) {
	test := assert.New(t)

	tcs := []struct {
		name              string
		envVars           map[string]string
		expectedAppConfig *AppConfig
		expectedError     error
	}{
		{
			name: "valid api config",
			envVars: map[string]string{
				"AWS_REGION":              "aws-region",
				"DB_ENDPOINT":             "http://localhost:8000",
				"DB_AWS_REGION":           "db-aws-region",
				"DB_TABLE_NAME":           "cluster-registry-local",
				"DB_INDEX_NAME":           "search-index-local",
				"SQS_ENDPOINT":            "http://localhost:9324",
				"SQS_AWS_REGION":          "sqs-aws-region",
				"SQS_QUEUE_NAME":          "cluster-registry-local",
				"OIDC_CLIENT_ID":          "oidc-client-id",
				"OIDC_ISSUER_URL":         "http://fake-oidc-provider",
				"API_RATE_LIMITER":        "enabled",
				"LOG_LEVEL":               "DEBUG",
				"API_HOST":                "custom-host:8080",
				"K8S_RESOURCE_ID":         "k8s-resource-id",
				"API_TENANT_ID":           "api-tenant-id",
				"API_CLIENT_ID":           "api-client-id",
				"API_CLIENT_SECRET":       "api-client-secret",
				"API_AUTHORIZED_GROUP_ID": "api-authorized-group-id",
			},
			expectedAppConfig: &AppConfig{
				ApiRateLimiterEnabled: true,
				ApiHost:               "custom-host:8080",
				AwsRegion:             "aws-region",
				DbEndpoint:            "http://localhost:8000",
				DbAwsRegion:           "db-aws-region",
				DbTableName:           "cluster-registry-local",
				DbIndexName:           "search-index-local",
				LogLevel:              log.DEBUG,
				OidcClientId:          "oidc-client-id",
				OidcIssuerUrl:         "http://fake-oidc-provider",
				SqsEndpoint:           "http://localhost:9324",
				SqsAwsRegion:          "sqs-aws-region",
				SqsQueueName:          "cluster-registry-local",
				SqsBatchSize:          10,
				SqsWaitSeconds:        5,
				SqsRunInterval:        30,
				K8sResourceId:         "k8s-resource-id",
				ApiTenantId:           "api-tenant-id",
				ApiClientId:           "api-client-id",
				ApiClientSecret:       "api-client-secret",
				ApiAuthorizedGroupId:  "api-authorized-group-id",
				ApiCacheTTL:           time.Hour,
				ApiCacheRedisHost:     "localhost:6379",
			},
			expectedError: nil,
		},
		{
			name: "invalid app config",
			envVars: map[string]string{
				"AWS_REGION":     "aws-region",
				"DB_ENDPOINT":    "http://localhost:8000",
				"DB_AWS_REGION":  "db-aws-region",
				"DB_TABLE_NAME":  "cluster-registry-local",
				"DB_INDEX_NAME":  "search-index-local",
				"SQS_ENDPOINT":   "http://localhost:9324",
				"SQS_AWS_REGION": "sqs-aws-region",
				"SQS_QUEUE_NAME": "cluster-registry-local",
				"OIDC_CLIENT_ID": "oidc-client-id",
			},
			expectedAppConfig: &AppConfig{
				ApiRateLimiterEnabled: true,
				ApiHost:               "custom-host:8080",
				AwsRegion:             "aws-region",
				DbEndpoint:            "http://localhost:8000",
				DbAwsRegion:           "db-aws-region",
				DbTableName:           "cluster-registry-local",
				DbIndexName:           "search-index-local",
				LogLevel:              log.DEBUG,
				OidcClientId:          "oidc-client-id",
				SqsEndpoint:           "http://localhost:9324",
				SqsAwsRegion:          "sqs-aws-region",
				SqsQueueName:          "cluster-registry-local",
				K8sResourceId:         "k8s-resource-id",
				ApiTenantId:           "api-tenant-id",
				ApiClientId:           "api-client-id",
				ApiClientSecret:       "api-client-secret",
				ApiAuthorizedGroupId:  "api-authorized-group-id",
			},
			expectedError: fmt.Errorf("environment variable OIDC_ISSUER_URL is not set"),
		},
	}

	for _, tc := range tcs {

		for k, v := range tc.envVars {
			_ = os.Setenv(k, v)
		}

		t.Logf("\tTest %s:\tWhen loading api environment variable", tc.name)

		appConfig, err := LoadApiConfig()

		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.Equal(tc.expectedAppConfig, appConfig)
		}

		for k := range tc.envVars {
			_ = os.Unsetenv(k)
		}
	}
}

func TestLoadClientConfig(t *testing.T) {
	test := assert.New(t)

	tcs := []struct {
		name              string
		envVars           map[string]string
		expectedAppConfig *AppConfig
		expectedError     error
	}{
		{
			name: "valid app config",
			envVars: map[string]string{
				"SQS_ENDPOINT":   "http://localhost:9324",
				"SQS_AWS_REGION": "sqs-aws-region",
				"SQS_QUEUE_NAME": "cluster-registry-local",
			},
			expectedAppConfig: &AppConfig{
				SqsEndpoint:  "http://localhost:9324",
				SqsAwsRegion: "sqs-aws-region",
				SqsQueueName: "cluster-registry-local",
			},
			expectedError: nil,
		},
		{
			name: "invalid app config",
			envVars: map[string]string{
				"SQS_ENDPOINT":   "http://localhost:9324",
				"SQS_AWS_REGION": "sqs-aws-region",
			},
			expectedAppConfig: &AppConfig{
				SqsEndpoint:  "http://localhost:9324",
				SqsAwsRegion: "sqs-aws-region",
			},
			expectedError: fmt.Errorf("environment variable SQS_QUEUE_NAME is not set"),
		},
	}

	for _, tc := range tcs {

		for k, v := range tc.envVars {
			_ = os.Setenv(k, v)
		}

		t.Logf("\tTest %s:\tWhen loading api environment variable", tc.name)

		appConfig, err := LoadClientConfig()

		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.Equal(tc.expectedAppConfig, appConfig)
		}

		for k := range tc.envVars {
			_ = os.Unsetenv(k)
		}
	}
}
