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
	"time"

	"github.com/labstack/gommon/log"
)

type AppConfig struct {
	ApiRateLimiterEnabled bool
	ApiHost               string
	AwsRegion             string
	DbEndpoint            string
	DbAwsRegion           string
	DbTableName           string
	DbIndexName           string
	LogLevel              log.Lvl
	OidcClientId          string
	OidcIssuerUrl         string
	SqsEndpoint           string
	SqsAwsRegion          string
	SqsQueueName          string
	K8sResourceId         string
	ApiTenantId           string
	ApiClientId           string
	ApiClientSecret       string
	ApiAuthorizedGroupId  string
	ApiCacheTTL           time.Duration
}

func LoadApiConfig() (*AppConfig, error) {
	awsRegion := getEnv("AWS_REGION", "")
	if awsRegion == "" {
		return nil, fmt.Errorf("environment variable AWS_REGION is not set")
	}

	dbAwsRegion := getEnv("DB_AWS_REGION", "")
	if dbAwsRegion == "" {
		return nil, fmt.Errorf("environment variable DB_AWS_REGION is not set")
	}

	dbEndpoint := getEnv("DB_ENDPOINT", "")
	if dbEndpoint == "" {
		return nil, fmt.Errorf("environment variable DB_ENDPOINT is not set")
	}

	dbTableName := getEnv("DB_TABLE_NAME", "")
	if dbTableName == "" {
		return nil, fmt.Errorf("environment variable DB_TABLE_NAME is not set")
	}

	dbIndexName := getEnv("DB_INDEX_NAME", "")

	sqsEndpoint := getEnv("SQS_ENDPOINT", "")
	if sqsEndpoint == "" {
		return nil, fmt.Errorf("environment variable SQS_ENDPOINT is not set")
	}

	sqsAwsRegion := getEnv("SQS_AWS_REGION", "")
	if sqsAwsRegion == "" {
		return nil, fmt.Errorf("environment variable SQS_AWS_REGION is not set")
	}

	sqsQueueName := getEnv("SQS_QUEUE_NAME", "")
	if sqsQueueName == "" {
		return nil, fmt.Errorf("environment variable SQS_QUEUE_NAME is not set")
	}

	oidcClientId := getEnv("OIDC_CLIENT_ID", "")
	if oidcClientId == "" {
		return nil, fmt.Errorf("environment variable OIDC_CLIENT_ID is not set")
	}

	oidcIssuerUrl := getEnv("OIDC_ISSUER_URL", "")
	if oidcIssuerUrl == "" {
		return nil, fmt.Errorf("environment variable OIDC_ISSUER_URL is not set")
	}

	apiRateLimiterEnabled := false
	configApiRateLimiter := getEnv("API_RATE_LIMITER", "")
	if configApiRateLimiter == "enabled" {
		apiRateLimiterEnabled = true
	}

	logLevel := log.WARN
	configLogLevel := getEnv("LOG_LEVEL", "INFO")
	if configLogLevel == "DEBUG" {
		logLevel = log.DEBUG
	} else if configLogLevel == "INFO" {
		logLevel = log.INFO
	}

	apiHost := getEnv("API_HOST", "0.0.0.0:8080")

	k8sResourceId := getEnv("K8S_RESOURCE_ID", "")
	if k8sResourceId == "" {
		return nil, fmt.Errorf("environment variable K8S_RESOURCE_ID is not set")
	}

	apiTenantId := getEnv("API_TENANT_ID", "")
	if apiTenantId == "" {
		return nil, fmt.Errorf("environment variable API_TENANT_ID is not set")
	}

	apiClientId := getEnv("API_CLIENT_ID", "")
	if apiClientId == "" {
		return nil, fmt.Errorf("environment variable API_CLIENT_ID is not set")
	}

	apiClientSecret := getEnv("API_CLIENT_SECRET", "")
	if apiClientSecret == "" {
		return nil, fmt.Errorf("environment variable API_CLIENT_SECRET is not set")
	}

	authorizedGroupId := getEnv("API_AUTHORIZED_GROUP_ID", "")
	if authorizedGroupId == "" {
		return nil, fmt.Errorf("environment variable API_AUTHORIZED_GROUP_ID is not set")
	}

	apiCacheTTLstr := getEnv("API_CACHE_TTL", "1h")
	apiCacheTTL, err := time.ParseDuration(apiCacheTTLstr)
	if err != nil {
		return nil, fmt.Errorf("error parsing API_CACHE_TTL: %v", err)
	}

	return &AppConfig{
		AwsRegion:             awsRegion,
		DbEndpoint:            dbEndpoint,
		DbAwsRegion:           dbAwsRegion,
		DbTableName:           dbTableName,
		DbIndexName:           dbIndexName,
		SqsEndpoint:           sqsEndpoint,
		SqsAwsRegion:          sqsAwsRegion,
		SqsQueueName:          sqsQueueName,
		OidcClientId:          oidcClientId,
		OidcIssuerUrl:         oidcIssuerUrl,
		ApiRateLimiterEnabled: apiRateLimiterEnabled,
		LogLevel:              logLevel,
		ApiHost:               apiHost,
		K8sResourceId:         k8sResourceId,
		ApiTenantId:           apiTenantId,
		ApiClientId:           apiClientId,
		ApiClientSecret:       apiClientSecret,
		ApiAuthorizedGroupId:  authorizedGroupId,
		ApiCacheTTL:           apiCacheTTL,
	}, nil
}

func LoadClientConfig() (*AppConfig, error) {
	sqsEndpoint := getEnv("SQS_ENDPOINT", "")
	if sqsEndpoint == "" {
		return nil, fmt.Errorf("environment variable SQS_ENDPOINT is not set")
	}

	sqsAwsRegion := getEnv("SQS_AWS_REGION", "")
	if sqsAwsRegion == "" {
		return nil, fmt.Errorf("environment variable SQS_AWS_REGION is not set")
	}

	sqsQueueName := getEnv("SQS_QUEUE_NAME", "")
	if sqsQueueName == "" {
		return nil, fmt.Errorf("environment variable SQS_QUEUE_NAME is not set")
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
