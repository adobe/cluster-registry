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
	"encoding/base64"
	"flag"
	"fmt"
	registryv1alpha1 "github.com/adobe/cluster-registry/pkg/api/registry/v1alpha1"
	"github.com/adobe/cluster-registry/pkg/client/controllers"
	"github.com/adobe/cluster-registry/pkg/config"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/client"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	configv1 "github.com/adobe/cluster-registry/pkg/api/config/v1"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/client/webhook"
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
	utilruntime.Must(registryv1alpha1.AddToScheme(scheme))
	utilruntime.Must(configv1.AddToScheme(scheme))
}

func main() {
	ctx := ctrl.SetupSignalHandler()

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
	var clientConfig configv1.ClientConfig
	clientConfigDefaults := configv1.ClientConfig{
		Namespace: namespace,
		AlertmanagerWebhook: configv1.AlertmanagerWebhookConfig{
			BindAddress: alertmanagerWebhookAddr,
			AlertMap:    []configv1.AlertRule{},
		},
		ServiceMetadata: configv1.ServiceMetadataConfig{
			WatchedGVKs:         []configv1.WatchedGVK{},
			ServiceIdAnnotation: "adobe.serviceid",
		},
	}
	options := ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
			ExtraHandlers: map[string]http.Handler{
				"/metrics/extra": promhttp.Handler(),
			},
		},
		HealthProbeBindAddress:     probeAddr,
		LeaderElection:             enableLeaderElection,
		LeaderElectionID:           "1d5078e3.registry.ethos.adobe.com",
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
	}

	if configFile != "" {
		options, clientConfig, err = apply(configFile, &clientConfigDefaults)
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

	appConfig, err := config.LoadClientConfig()

	if err != nil {
		setupLog.Error(err, "failed to load client configuration")
		os.Exit(1)
	}

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

	if err = (&controllers.ServiceMetadataWatcherReconciler{
		Client:              mgr.GetClient(),
		Log:                 ctrl.Log.WithName("controllers").WithName("ServiceMetadataWatcher"),
		Scheme:              mgr.GetScheme(),
		WatchedGVKs:         loadWatchedGVKs(clientConfig),
		ServiceIdAnnotation: clientConfig.ServiceMetadata.ServiceIdAnnotation,
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceMetadataWatcher")
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
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running cluster-registry-client")
		os.Exit(1)
	}
}

func loadWatchedGVKs(cfg configv1.ClientConfig) []schema.GroupVersionKind {
	availableGVKs, err := getAvailableGVKs()
	if err != nil {
		return []schema.GroupVersionKind{}
	}
	var GVKs []schema.GroupVersionKind
	for _, gvk := range cfg.ServiceMetadata.WatchedGVKs {
		gvk := schema.GroupVersionKind{
			Group:   gvk.Group,
			Version: gvk.Version,
			Kind:    gvk.Kind,
		}
		if _, found := availableGVKs[gvk]; !found {
			setupLog.Info("GVK not installed in the cluster", "gvk", gvk)
			continue
		}
		GVKs = append(GVKs, gvk)
	}
	return GVKs
}

func apply(configFile string, clientConfigDefaults *configv1.ClientConfig) (ctrl.Options, configv1.ClientConfig, error) {
	options, cfg, err := configv1.Load(scheme, configFile, clientConfigDefaults)
	if err != nil {
		return options, cfg, err
	}

	cfgStr, err := configv1.Encode(scheme, &cfg)
	if err != nil {
		return options, cfg, err
	}
	setupLog.Info("Successfully loaded configuration", "config", cfgStr)

	return options, cfg, nil
}

func getAvailableGVKs() (map[schema.GroupVersionKind]bool, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		return nil, fmt.Errorf("unable to create discovery client: %w", err)
	}

	_, availableResources, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		return nil, fmt.Errorf("unable to get available API resources: %w", err)
	}

	availableGVKs := make(map[schema.GroupVersionKind]bool)
	for _, list := range availableResources {
		groupVersion, _ := schema.ParseGroupVersion(list.GroupVersion)
		for _, resource := range list.APIResources {
			gvk := schema.GroupVersionKind{
				Group:   groupVersion.Group,
				Version: groupVersion.Version,
				Kind:    resource.Kind,
			}
			availableGVKs[gvk] = true
		}
	}

	return availableGVKs, nil
}
