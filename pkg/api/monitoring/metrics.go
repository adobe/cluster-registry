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

package monitoring

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultMetricPath = "/metrics"
	defaultSubsystem  = "echo"
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

/*
RequestCounterLabelMappingFunc is a function which can be supplied to the middleware to control
the cardinality of the request counter's "url" label, which might be required in some contexts.
For instance, if for a "/customer/:name" route you don't want to generate a time series for every
possible customer name, you could use this function:

func(c echo.Context) string {
	url := c.Request.URL.Path
	for _, p := range c.Params {
		if p.Key == "name" {
			url = strings.Replace(url, p.Value, ":name", 1)
			break
		}
	}
	return url
}

which would map "/customer/alice" and "/customer/bob" to their template "/customer/:name".
It can also be applied for the "Host" label
*/
type requestCounterLabelMappingFunc func(c echo.Context) string

// Metric is a definition for the name, description, type, ID, and
// prometheus.Collector type (i.e. CounterVec, HistrogramVec, etc) of each metric
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
	Use(e *echo.Echo)
}

// Metrics contains the metrics gathered by the instance and its path
type Metrics struct {
	ingressReqCnt, egressReqCnt *prometheus.CounterVec
	ingressReqDur, egressReqDur *prometheus.HistogramVec

	metricsList []*Metric
	metricsPath string
	subsystem   string
	skipper     middleware.Skipper
	isUnitTest  bool

	requestCounterURLLabelMappingFunc requestCounterLabelMappingFunc

	// Context string to use as a prometheus URL label
	urlLabelFromContext string
}

// NewMetrics generates a new set of metrics with a certain subsystem name
func NewMetrics(subsystem string, skipper middleware.Skipper, isUnitTest bool) *Metrics {
	var metricsList []*Metric
	if skipper == nil {
		skipper = middleware.DefaultSkipper
	}

	metricsList = append(metricsList, ingressMetrics...)
	metricsList = append(metricsList, egressMetrics...)

	m := &Metrics{
		metricsList: metricsList,
		metricsPath: defaultMetricPath,
		subsystem:   defaultSubsystem,
		skipper:     skipper,
		isUnitTest:  isUnitTest,
		requestCounterURLLabelMappingFunc: func(c echo.Context) string {
			return c.Path() // i.e. by default do nothing, i.e. return URL as is
		},
	}

	m.registerMetrics(subsystem)

	return m
}

func prometheusHandler() echo.HandlerFunc {
	h := promhttp.Handler()
	return func(c echo.Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
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
		}
		metricDef.MetricCollector = metric
	}
}

// Use adds the middleware to the Echo engine.
func (m *Metrics) Use(e *echo.Echo) {
	e.Use(m.handlerFunc)
	e.GET(m.metricsPath, prometheusHandler())
}

// HandlerFunc defines handler function for middleware
func (m *Metrics) handlerFunc(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Path() == m.metricsPath {
			return next(c)
		}
		if m.skipper(c) {
			return next(c)
		}

		start := time.Now()
		err := next(c)
		elapsed := float64(time.Since(start)) / float64(time.Second)

		method := c.Request().Method
		status := c.Response().Status
		if err != nil {
			var httpError *echo.HTTPError
			if errors.As(err, &httpError) {
				status = httpError.Code
			}
			if status == 0 || status == http.StatusOK {
				status = http.StatusInternalServerError
			}
		}

		url := m.requestCounterURLLabelMappingFunc(c)
		if len(m.urlLabelFromContext) > 0 {
			u := c.Get(m.urlLabelFromContext)
			if u == nil {
				u = "unknown"
			}
			url = u.(string)
		}

		statusStr := strconv.Itoa(status)
		m.RecordIngressRequestCnt(statusStr, method, url)
		m.RecordIngressRequestDur(statusStr, method, url, elapsed)

		return err
	}
}

// RecordEgressRequestCnt increases the Egress counter for a taget
func (m *Metrics) RecordEgressRequestCnt(target string) {
	m.egressReqCnt.WithLabelValues(target).Inc()
}

// RecordEgressRequestDur registers the Egress duration for a taget
func (m *Metrics) RecordEgressRequestDur(target string, elapsed float64) {
	m.egressReqDur.WithLabelValues(target).Observe(elapsed)
}

// RecordIngressRequestCnt increases the Ingress counter for a taget
func (m *Metrics) RecordIngressRequestCnt(code, method, url string) {
	m.ingressReqCnt.WithLabelValues(code, method, url).Inc()
}

// RecordIngressRequestDur registers the Egress duration for a taget
func (m *Metrics) RecordIngressRequestDur(code, method, url string, elapsed float64) {
	m.ingressReqDur.WithLabelValues(code, method, url).Observe(elapsed)
}
