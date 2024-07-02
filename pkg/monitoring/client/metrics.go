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
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsI interface
type MetricsI interface {
	RecordEgressRequestCnt(target string)
	RecordEgressRequestDur(target string, elapsed float64)
	RecordDMSLastTimestamp()
	GetMetricByName(name string) prometheus.Collector
}

// Metrics contains the metrics gathered by the instance and its path
type Metrics struct {
	egressReqCnt     *prometheus.CounterVec
	egressReqDur     *prometheus.HistogramVec
	dmsLastTimestamp prometheus.Gauge
	metrics          []prometheus.Collector
}

// NewMetrics func
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Init func
func (m *Metrics) Init(isUnitTest bool) {
	reg := prometheus.DefaultRegisterer
	if isUnitTest {
		reg = prometheus.NewRegistry()
	}

	var egressReqCnt prometheus.Collector = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "cluster_registry_cc_egress_requests_total",
			Help: "How many egress requests sent, partitioned by target.",
		},
		[]string{"target"},
	)
	m.egressReqCnt = egressReqCnt.(*prometheus.CounterVec)
	m.metrics = append(m.metrics, m.egressReqCnt)

	var egressReqDur prometheus.Collector = promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "cluster_registry_cc_egress_request_duration_seconds",
			Help: "The Egress HTTP request latencies in seconds partitioned by target.",
		},
		[]string{"target"},
	)
	m.egressReqDur = egressReqDur.(*prometheus.HistogramVec)
	m.metrics = append(m.metrics, m.egressReqDur)

	var dmsLastTimestamp prometheus.Collector = promauto.With(reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "cluster_registry_cc_deadmansswitch_last_timestamp_seconds",
			Help: "Last timestamp when a DeadMansSwitch alert was received.",
		},
	)
	m.dmsLastTimestamp = dmsLastTimestamp.(prometheus.Gauge)
	m.metrics = append(m.metrics, m.dmsLastTimestamp)
}

// RecordEgressRequestCnt increases the Egress counter for a taget
func (m *Metrics) RecordEgressRequestCnt(target string) {
	m.egressReqCnt.WithLabelValues(target).Inc()
}

// RecordEgressRequestDur registers the Egress duration for a taget
func (m *Metrics) RecordEgressRequestDur(target string, elapsed float64) {
	m.egressReqDur.WithLabelValues(target).Observe(elapsed)
}

// RecordDMSLastTimestamp records the current timestamp when a DMS is received
func (m *Metrics) RecordDMSLastTimestamp() {
	m.dmsLastTimestamp.SetToCurrentTime()
}

// GetMetricByName returns a metric by it's fully qualified name (used for testing purposes)
func (m *Metrics) GetMetricByName(name string) prometheus.Collector {
	// don't judge me :(
	desc := make(chan *prometheus.Desc, 1)
	for _, metric := range m.metrics {
		metric.Describe(desc)
		re, err := regexp.Compile(`Desc{fqName: "([a-z_]+)"`)
		if err != nil {
			continue
		}
		fqName := re.FindStringSubmatch((*<-desc).String())[1]
		if fqName == name {
			return metric
		}
	}

	return nil
}
