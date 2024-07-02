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
	"flag"
	"fmt"
	configv1 "github.com/adobe/cluster-registry/pkg/api/config/v1"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	registryv1alpha1 "github.com/adobe/cluster-registry/pkg/api/registry/v1alpha1"
	"github.com/adobe/cluster-registry/pkg/config"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/client"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"github.com/adobe/cluster-registry/pkg/sync/manager"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
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
	var probeAddr string
	var namespace string
	var enableLeaderElection bool

	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":9090", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":9091", "The address the probe endpoint binds to.")
	flag.StringVar(&namespace, "namespace", "cluster-registry", "The namespace where cluster-registry-sync-manager will run.")
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
	var syncConfig configv1.SyncConfig
	syncConfigDefaults := configv1.SyncConfig{
		Namespace:   namespace,
		WatchedGVKs: []configv1.WatchedGVK{},
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
		LeaderElectionID:           "sync.registry.ethos.adobe.com",
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
	}

	if configFile != "" {
		options, syncConfig, err = apply(options, configFile, &syncConfigDefaults)
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}
	setupLog.Info("using client configuration", "config", syncConfig)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start cluster-registry-sync-manager")
		os.Exit(1)
	}

	m := monitoring.NewMetrics()
	m.Init(false)

	appConfig, err := config.LoadClientConfig()

	if err != nil {
		setupLog.Error(err, "failed to load client configuration")
		os.Exit(1)
	}

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
		setupLog.Error(err, "cannot create SQS client")
		os.Exit(1)
	}

	if err = (&manager.SyncController{
		Client:      mgr.GetClient(),
		Log:         ctrl.Log.WithName("controllers").WithName("SyncController"),
		Scheme:      mgr.GetScheme(),
		WatchedGVKs: loadWatchedGVKs(syncConfig),
		Queue:       q,
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SyncController")
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

	setupLog.Info("starting cluster-sync-manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running cluster-registry-sync-manager")
		os.Exit(1)
	}
}

func loadWatchedGVKs(cfg configv1.SyncConfig) []schema.GroupVersionKind {
	availableGVKs, err := getAvailableGVKs()
	if err != nil {
		return []schema.GroupVersionKind{}
	}
	var GVKs []schema.GroupVersionKind
	for _, gvk := range cfg.WatchedGVKs {
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

func apply(defaultOptions ctrl.Options, configFile string, syncConfigDefaults *configv1.SyncConfig) (ctrl.Options, configv1.SyncConfig, error) {
	options, cfg, err := configv1.NewSyncConfig(defaultOptions, scheme, configFile, syncConfigDefaults)
	if err != nil {
		return options, cfg, err
	}

	cfgStr, err := cfg.Encode(scheme)
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
