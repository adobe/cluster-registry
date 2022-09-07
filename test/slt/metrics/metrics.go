package metrics

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

var logger echo.Logger

var E2eStatus = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "cr_e2e_status",
		Help: "The status of the last SLT, has values between 1 if the check passed and 0 " +
			"if it failed.",
	},
)

var E2eDuration = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "cr_e2e_duration",
		Help: "It's how much time did the last e2e test take to run (in seconds). Be mindful " +
			"that the time between the crd is updated and the change propagates to the " +
			"API is around 11s, so the full slt duration can't be smaller than that.",
	},
)

var E2eProcessingDuration = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "cr_e2e_processing_duration",
		Help: "It's how much time did the last e2e test take to run (in seconds). But the time " +
			"waited(sleep) for the update to propagate to the API is subtracted, so this calculates " +
			"only the time the code is running without the sleeps.",
	},
)

var ClusterReqDuration = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "cr_cluster_request_duration",
		Help: "It's how much time took to get a cluster (in seconds) using the " +
			"/clusters/[clustername] endpoint.",
	},
)

var AllClustersReqDuration = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "cr_all_clusters_request_duration",
		Help: "It's how much time took to get a page of 200 clusters (in seconds) using the /clusters endpoint.",
	},
)

// SetLogger sets the global logger
func SetLogger(lgr echo.Logger) {
	logger = lgr
}

// RegisterMetrics registers the metrics from this package
func RegisterMetrics() {
	if err := prometheus.Register(E2eStatus); err != nil {
		logger.Fatalf("Error registering metric: %s", err)
	}

	if err := prometheus.Register(E2eDuration); err != nil {
		logger.Fatalf("Error registering metric: %s", err)
	}

	if err := prometheus.Register(E2eProcessingDuration); err != nil {
		logger.Fatalf("Error registering metric: %s", err)
	}

	if err := prometheus.Register(ClusterReqDuration); err != nil {
		logger.Fatalf("Error registering metric: %s", err)
	}

	if err := prometheus.Register(AllClustersReqDuration); err != nil {
		logger.Fatalf("Error registering metric: %s", err)
	}
}
