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
	"errors"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"github.com/adobe/cluster-registry/pkg/sync/client"
	"github.com/adobe/cluster-registry/pkg/sync/event"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	rootCmd = &cobra.Command{
		Use:              "cluster-registry-sync-client",
		Short:            "Cluster Registry Sync Client is a service that keep the Cluster CRD in sync",
		Long:             "Cluster Registry Sync Client is a service that creates or updates the cluster CRD based on the messages received from the Cluster Registry Sync manager",
		PersistentPreRun: loadAppConfig,
		Run:              run,
	}

	logLevel, logFormat    string
	appConfig              *config.AppConfig
	namespace              string
	healthProbeBindAddress string
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", logrus.DebugLevel.String(), "The verbosity level of the logs, can be [panic|fatal|error|warn|info|debug|trace]")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "The output format of the logs, can be [text|json]")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "cluster-registry", "The namespace where cluster-registry-sync-client will run.")
	rootCmd.PersistentFlags().StringVar(&healthProbeBindAddress, "health-probe-bind-address", ":8080", "The address the health probes will bind to.")
}

func loadAppConfig(cmd *cobra.Command, args []string) {
	client.InitLogger(logLevel, logFormat)

	log.Info("Loading the configuration")

	var err error
	appConfig, err = config.LoadSQSConfig()
	if err != nil {
		log.Error("Cannot load the cluster-registry-sync-client configuration: ", err.Error())
		os.Exit(1)
	}

	log.Info("Config loaded successfully")
}

func run(cmd *cobra.Command, args []string) {
	ctx := signals.SetupSignalHandler()

	q, err := sqs.NewSQS(sqs.Config{
		AWSRegion:         appConfig.SqsAwsRegion,
		Endpoint:          appConfig.SqsEndpoint,
		QueueName:         appConfig.SqsQueueName,
		BatchSize:         appConfig.SqsBatchSize,
		VisibilityTimeout: appConfig.SqsVisibilityTimeout,
		WaitSeconds:       appConfig.SqsWaitSeconds,
		RunInterval:       appConfig.SqsRunInterval,
		RunOnce:           false,
		MaxHandlers:       appConfig.SqsMaxHandlers,
		BusyTimeout:       appConfig.SqsBusyTimeout,
	})
	if err != nil {
		log.Panicf("Error while trying to create SQS client: %v", err.Error())
	}

	dynamicClient, err := client.GetDynamicClientSet()
	if err != nil {
		log.Error("Error while trying to create dynamic client: ", err.Error())
		os.Exit(1)
	}

	handler := event.NewPartialClusterUpdateHandler(dynamicClient, namespace)
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

	go serveHealthProbes(ctx.Done(), healthProbeBindAddress)

	log.Info("Starting the Cluster Registry Sync Client")
	q.Poll()
}

func serveHealthProbes(stop <-chan struct{}, healthProbeBindAddress string) {
	healthzHandler := &healthz.Handler{Checks: map[string]healthz.Checker{
		"healthz": healthz.Ping,
	}}
	readyzHandler := &healthz.Handler{Checks: map[string]healthz.Checker{
		"readyz": healthz.Ping,
	}}

	mux := http.NewServeMux()
	mux.Handle("/readyz", http.StripPrefix("/readyz", readyzHandler))
	mux.Handle("/healthz", http.StripPrefix("/healthz", healthzHandler))

	server := http.Server{
		Handler: mux,
	}

	ln, err := net.Listen("tcp", healthProbeBindAddress)
	if err != nil {
		log.Errorf("error listening on %s: %v", healthProbeBindAddress, err)
		return
	}

	log.Infof("Health probes listening on %s", healthProbeBindAddress)
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-stop
	if err := server.Shutdown(context.Background()); err != nil {
		klog.Fatal(err)
	}
}

func main() {
	Execute()
}
