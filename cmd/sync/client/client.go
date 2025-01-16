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
	"github.com/adobe/cluster-registry/pkg/sync/event"
	"os"

	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"github.com/adobe/cluster-registry/pkg/sync/client"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:              "cluster-registry-sync-client",
		Short:            "Cluster Registry Sync Client is a service that keep the Cluster CRD in sync",
		Long:             "Cluster Registry Sync Client is a service that creates or updates the cluster CRD based on the messages received from the Cluster Registry Sync manager",
		PersistentPreRun: loadAppConfig,
		Run:              run,
	}

	logLevel, logFormat string
	appConfig           *config.AppConfig
	namespace           string
	cfgFile             string
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "The path to the yaml configuration file")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", logrus.DebugLevel.String(), "The verbosity level of the logs, can be [panic|fatal|error|warn|info|debug|trace]")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "The output format of the logs, can be [text|json]")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "cluster-registry", "The namespace where cluster-registry-sync-client will run.")
	err := rootCmd.MarkPersistentFlagRequired("config")
	if err != nil {
		log.Fatalln("No config flag configured")
	}
}

func loadAppConfig(cmd *cobra.Command, args []string) {
	client.InitLogger(logLevel, logFormat)

	log.Info("Loading the configuration")

	var err error
	appConfig, err = config.LoadSyncClientConfig()
	if err != nil {
		log.Error("Cannot load the cluster-registry-sync-client configuration:", err.Error())
		os.Exit(1)
	}

	log.Info("Config loaded successfully")
}

func run(cmd *cobra.Command, args []string) {
	log.Info("Cluster Registry Sync Client is running")

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

	handler := event.NewPartialClusterUpdateHandler()
	q.RegisterHandler(func(msg *awssqs.Message) {
		log.Debugf("Received message: %s", *msg.MessageId)
		e, err := sqs.NewEvent(msg)
		if err != nil {
			log.Errorf("Cannot create event from message: %s", err.Error())
			return
		}
		if e.Type != sqs.PartialClusterUpdateEvent {
			log.Infof("Not interested in event of type %s, skipping", e.Type)
			return
		}
		log.Debugf("Handling event for message: %s", *msg.MessageId)
		if err = handler.Handle(e); err != nil {
			log.Errorf("Failed to handle event: %s", err.Error())
			return
		}
		if err = q.Delete(msg); err != nil {
			log.Errorf("Failed to delete message: %s", err.Error())
			return
		}
	})

	log.Info("Starting the Cluster Registry Sync Client")

	q.Poll()
}

func main() {
	Execute()
}
