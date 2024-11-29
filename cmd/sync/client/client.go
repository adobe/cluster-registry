package main

import (
	"fmt"
	"os"

	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/sqs"
	client "github.com/adobe/cluster-registry/pkg/sync/client"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	logLevel, logFormat string
	appConfig           *config.AppConfig
	namespace           string
	//clusterName         string
	//cfgFile             string
)

func InitCLI() *cobra.Command {

	var rootCmd = &cobra.Command{
		Use:              "cluster-registry-client-sync",
		Short:            "Cluster Registry Sync Client is a service that keep in sync the cluster CRD",
		Long:             "\nCluster Registry Sync Client is a service that creates or updates the cluster CRD based on the messages received from the Cluster Registry Sync manager",
		PersistentPreRun: loadAppConfig,
		Run:              run,
	}

	initFlags(rootCmd)

	return rootCmd
}

func initFlags(rootCmd *cobra.Command) {

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", logrus.DebugLevel.String(), "The verbosity level of the logs, can be [panic|fatal|error|warn|info|debug|trace]")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "The output format of the logs, can be [text|json]")
	//rootCmd.PersistentFlags().StringVar(&cfgFile, "config-file", "", "The path to the configuration file")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "cluster-registry", "The namespace where cluster-registry-sync-client will run.")
}

func loadAppConfig(cmd *cobra.Command, args []string) {

	client.InitLogger(logLevel, logFormat)

	log.Info("Starting the Cluster Registry Sync Client")

	log.Info("Loading the configuration")
	appConfig, err := config.LoadSyncClientConfig()
	if err != nil {
		log.Error("Cannot load the cluster-registry-sync-client configuration:", err.Error())
		os.Exit(1)
	}
	log.Info("Config loaded successfully")
	log.Info("Cluster (custom resource) to be checked:", appConfig.ClusterName)
}

func run(cmd *cobra.Command, args []string) {

	log.Info("Cluster Registry Sync Client is running")

	// Consume the messages from the queue using a sync consumer
	sqsInstance := sqs.NewSQS(appConfig)
	log.Info("Starting the SQS sync consumer")

	handler := func(m *awssqs.Message) error {
		spew.Dump(m)
		// TODO
		return nil
	}
	syncConsumer := sqs.NewSyncConsumer(sqsInstance, appConfig, handler)
	go syncConsumer.Consume()

	// Block the thread
	select {}
}

func main() {

	rootCmd := InitCLI()

	// Execute the CLI application
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//TODO

}
