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

package monitoring

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Ingress metrics
var ingressReqCnt = &Metric{
	ID:          "ingresReqCnt",
	Name:        "ingress_requests_total",
	Description: "How many HTTP requests processed, partitioned by status code, HTTP method and url.",
	Type:        "counter_vec",
	Args:        []string{"code", "method", "url"}}

var ingressReqDur = &Metric{
	ID:          "ingressReqDur",
	Name:        "ingress_request_duration_seconds",
	Description: "The HTTP request latencies in seconds partitioned by status code, HTTP method and url.",
	Args:        []string{"code", "method", "url"},
	Type:        "histogram_vec"}

var ingressMetrics = []*Metric{
	ingressReqCnt,
	ingressReqDur,
}

// EgressMetrics
var egressReqCnt = &Metric{
	ID:          "egressReqCnt",
	Name:        "egress_requests_total",
	Description: "How many egress requests sent, partitioned by target.",
	Type:        "counter_vec",
	Args:        []string{"target"}}

var egressReqDur = &Metric{
	ID:          "egressReqDur",
	Name:        "egress_request_duration_seconds",
	Description: "The Egress HTTP request latencies in seconds partitioned by target.",
	Args:        []string{"target"},
	Type:        "histogram_vec"}

var egressMetrics = []*Metric{
	egressReqCnt,
	egressReqDur,
}

var errCnt = &Metric{
	ID:          "ErrCnt",
	Name:        "error_count",
	Description: "The total number of errors, partitioned by target.",
	Type:        "counter_vec",
	Args:        []string{"target"},
}

// Metric is a definition for the name, description, type, ID, and
// prometheus.Collector type (i.e. CounterVec, HistogramVec, etc) of each metric
type Metric struct {
	MetricCollector prometheus.Collector
	ID              string
	Name            string
	Description     string
	Type            string
	Args            []string
}

// MetricsI interface
type MetricsI interface {
	RecordEgressRequestCnt(target string)
	RecordEgressRequestDur(target string, elapsed float64)
	RecordIngressRequestCnt(code, method, url string)
	RecordIngressRequestDur(code, method, url string, elapsed float64)
	RecordErrorCnt(target string)
	Use(e *echo.Echo)
}

// Metrics contains the metrics gathered by the instance and its path
type Metrics struct {
	ingressReqCnt *prometheus.CounterVec
	egressReqCnt  *prometheus.CounterVec
	ingressReqDur *prometheus.HistogramVec
	egressReqDur  *prometheus.HistogramVec
	errCnt        *prometheus.CounterVec

	metricsList []*Metric
	subsystem   string
	isUnitTest  bool

	// Context string to use as a prometheus URL label
	URLLabelFromContext string
}

// NewMetrics generates a new set of metrics with a certain subsystem name
func NewMetrics(subsystem string, isUnitTest bool) *Metrics {
	var metricsList []*Metric

	metricsList = append(metricsList, ingressMetrics...)
	metricsList = append(metricsList, egressMetrics...)
	metricsList = append(metricsList, errCnt)

	m := &Metrics{
		metricsList: metricsList,
		subsystem:   subsystem,
		isUnitTest:  isUnitTest,
	}

	m.registerMetrics(subsystem)

	return m
}

func (m *Metrics) registerMetrics(subsystem string) {
	var metric prometheus.Collector

	reg := prometheus.DefaultRegisterer
	if m.isUnitTest {
		reg = prometheus.NewRegistry()
	}

	factory := promauto.With(reg)

	for _, metricDef := range m.metricsList {
		switch metricDef.Type {
		case "counter_vec":
			metric = factory.NewCounterVec(
				prometheus.CounterOpts{
					Subsystem: subsystem,
					Name:      metricDef.Name,
					Help:      metricDef.Description,
				},
				metricDef.Args,
			)
		case "histogram_vec":
			metric = factory.NewHistogramVec(
				prometheus.HistogramOpts{
					Subsystem: subsystem,
					Name:      metricDef.Name,
					Help:      metricDef.Description,
				},
				metricDef.Args,
			)
		}

		switch metricDef {
		case ingressReqCnt:
			m.ingressReqCnt = metric.(*prometheus.CounterVec)
		case ingressReqDur:
			m.ingressReqDur = metric.(*prometheus.HistogramVec)
		case egressReqCnt:
			m.egressReqCnt = metric.(*prometheus.CounterVec)
		case egressReqDur:
			m.egressReqDur = metric.(*prometheus.HistogramVec)
		case errCnt:
			m.errCnt = metric.(*prometheus.CounterVec)
		}
		metricDef.MetricCollector = metric
	}
}

// RecordErrorCnt increases the error counter for a target
func (m *Metrics) RecordErrorCnt(target string) {
	m.errCnt.WithLabelValues(target).Inc()
}

// RecordEgressRequestCnt increases the Egress counter for a target
func (m *Metrics) RecordEgressRequestCnt(target string) {
	m.egressReqCnt.WithLabelValues(target).Inc()
}

// RecordEgressRequestDur registers the Egress duration for a target
func (m *Metrics) RecordEgressRequestDur(target string, elapsed float64) {
	m.egressReqDur.WithLabelValues(target).Observe(elapsed)
}

// RecordIngressRequestCnt increases the Ingress counter for a target
func (m *Metrics) RecordIngressRequestCnt(code, method, url string) {
	m.ingressReqCnt.WithLabelValues(code, method, url).Inc()
}

// RecordIngressRequestDur registers the Egress duration for a target
func (m *Metrics) RecordIngressRequestDur(code, method, url string, elapsed float64) {
	m.ingressReqDur.WithLabelValues(code, method, url).Observe(elapsed)
}
