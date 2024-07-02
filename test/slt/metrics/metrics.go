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

package metrics

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

var logger echo.Logger

const metricsPrefix = "cluster_registry_slt"

var reqDurBuckets = []float64{.01, .05, .1, .2, .3, .5, .75, 1, 3, 5, 10, 15, 20, 40, 60}

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
