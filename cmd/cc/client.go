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

package main

import (
	"flag"
	"os"

	"github.com/adobe/cluster-registry/pkg/api/sqs"
	"github.com/adobe/cluster-registry/pkg/api/utils"
	"github.com/adobe/cluster-registry/pkg/cc/controllers"
	"github.com/adobe/cluster-registry/pkg/cc/monitoring"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"encoding/base64"

	configv1 "github.com/adobe/cluster-registry/pkg/cc/api/config/v1"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/cc/webhook"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(registryv1.AddToScheme(scheme))
	utilruntime.Must(configv1.AddToScheme(scheme))
}

func main() {
	var configFile string
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var alertmanagerWebhookAddr string
	var namespace string

	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":9090", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":9091", "The address the probe endpoint binds to.")
	flag.StringVar(&alertmanagerWebhookAddr, "alertmanager-webhook-bind-address", ":9092", "The address the alertmanager webhook endpoint binds to.")
	flag.StringVar(&namespace, "namespace", "cluster-registry", "The namespace where cluster-registry-client will run.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		// TODO: change this to false
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var err error
	clientConfig := configv1.ClientConfig{
		Namespace: namespace,
		AlertmanagerWebhook: configv1.AlertmanagerWebhookConfig{
			BindAddress: alertmanagerWebhookAddr,
			AlertMap:    []configv1.AlertRule{},
		},
	}
	options := ctrl.Options{
		Scheme:                 scheme,
		Namespace:              namespace,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "0c4967d2.registry.ethos.adobe.com",
	}

	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&clientConfig))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}
	setupLog.Info("using client configuration", "config", clientConfig)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start cluster-registry-client")
		os.Exit(1)
	}

	m := monitoring.NewMetrics()
	m.Init(false)

	// InClusterConfiguration doesn't initialize the CAData field by default
	// for some reason, so we're doing this manually by calling LoadTLSFiles
	if err = rest.LoadTLSFiles(mgr.GetConfig()); err != nil {
		setupLog.Error(err, "failed to load TLS files")
		os.Exit(1)
	}

	appConfig := utils.LoadClientConfig()
	sqsProducer := sqs.NewProducer(appConfig, m)

	if err = (&controllers.ClusterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Cluster"),
		Scheme: mgr.GetScheme(),
		Queue:  sqsProducer,
		CAData: base64.StdEncoding.EncodeToString(mgr.GetConfig().CAData),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := mgr.AddMetricsExtraHandler("/metrics/extra", promhttp.Handler()); err != nil {
		setupLog.Error(err, "unable to set up extra metrics handler")
		os.Exit(1)
	}

	go func() {
		setupLog.Info("starting alertmanager webhook server", "addr", clientConfig.AlertmanagerWebhook.BindAddress)
		if err := (&webhook.Server{
			Client:      mgr.GetClient(),
			Namespace:   clientConfig.Namespace,
			BindAddress: clientConfig.AlertmanagerWebhook.BindAddress,
			Log:         ctrl.Log.WithName("webhook"),
			Metrics:     m,
			AlertMap:    clientConfig.AlertmanagerWebhook.AlertMap,
		}).Start(); err != nil {
			setupLog.Error(err, "unable to start alertmanager webhook server")
			os.Exit(1)
		}
	}()

	setupLog.Info("starting cluster-registry-client")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running cluster-registry-client")
		os.Exit(1)
	}
}
