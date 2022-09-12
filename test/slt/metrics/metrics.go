package metrics

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

var logger echo.Logger

const metricsPrefix = "cluster_registry_slt"

var reqDurBuckets = []float64{.25, .5, 1, 3, 5, 10, 15, 20}

var TestStatus = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: metricsPrefix,
		Name:      "test_status",
		Help: "The status of the last SLT, has values between 1 if the check passed and 0 " +
			"if it failed.",
	},
	[]string{"test"},
)

var EgressReqDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: metricsPrefix,
		Name:      "egress_request_duration_seconds",
		Help:      "The duration of the requests to the Cluster Registry API.",
		Buckets:   reqDurBuckets,
	},
	[]string{"route", "method", "status_code"},
)

var TestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: metricsPrefix,
		Name:      "test_duration_seconds",
		Help: "It's how much time took for the test to complete. For the e2e test the " +
			"sleeps are excluded.",
		Buckets: reqDurBuckets,
	},
	[]string{"test"},
)

var ErrCnt = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: metricsPrefix,
		Name:      "error_count",
		Help:      "The total number of errors, labeled by the test",
	},
	[]string{"test"},
)

// SetLogger sets the global logger
func SetLogger(lgr echo.Logger) {
	logger = lgr
}

// RegisterMetrics registers the metrics from this package
func RegisterMetrics() {
	if err := prometheus.Register(TestStatus); err != nil {
		logger.Fatalf("Error registering metric: %s", err)
	}

	if err := prometheus.Register(EgressReqDuration); err != nil {
		logger.Fatalf("Error registering metric: %s", err)
	}

	if err := prometheus.Register(TestDuration); err != nil {
		logger.Fatalf("Error registering metric: %s", err)
	}

	if err := prometheus.Register(ErrCnt); err != nil {
		logger.Fatalf("Error registering metric: %s", err)
	}
}
