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
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	egressTarget              = "testing_egress"
	expectedMetricsRegistered = 5
	ingressCode               = "200"
	ingressMethod             = "GET"
	ingressURL                = "/api/v1/clusters/:name"
	minRand                   = 1
	maxRand                   = 2.5
	subsystem                 = "testing"
)

func TestNewMetrics(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics(subsystem, true)
	test.NotNil(m)
}

func TestMetricsRegistered(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics(subsystem, true)

	test.Equal(len(m.metricsList), expectedMetricsRegistered)
}

func TestRecordStatusErrorCnt(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics(subsystem, true)

	m.RecordErrorCnt(egressTarget)

	test.Equal(1, testutil.CollectAndCount(*m.errCnt))
	test.Equal(float64(1), testutil.ToFloat64((*m.errCnt).WithLabelValues(egressTarget)))
}

func TestRecordEgressRequestCnt(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics(subsystem, true)

	m.RecordEgressRequestCnt(egressTarget)

	test.Equal(1, testutil.CollectAndCount(*m.egressReqCnt))
	test.Equal(float64(1), testutil.ToFloat64((*m.egressReqCnt).WithLabelValues(egressTarget)))
}

// Generate a random float number between min and max
func generateFloatRand(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func TestRecordEgressRequestDur(t *testing.T) {
	m := NewMetrics(subsystem, true)

	randomFloat := generateFloatRand(minRand, maxRand)
	m.RecordEgressRequestDur(egressTarget, randomFloat)

	expected := fmt.Sprintf(`
		# HELP %[1]s_egress_request_duration_seconds The Egress HTTP request latencies in seconds partitioned by target.
		# TYPE %[1]s_egress_request_duration_seconds histogram
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="0.005"} 0
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="0.01"} 0
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="0.025"} 0
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="0.05"} 0
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="0.1"} 0
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="0.25"} 0
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="0.5"} 0
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="1"} 0
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="2.5"} 1
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="5"} 1
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="10"} 1
		%[1]s_egress_request_duration_seconds_bucket{target="%[2]s",le="+Inf"} 1
		%[1]s_egress_request_duration_seconds_sum{target="%[2]s"} %[3]s
		%[1]s_egress_request_duration_seconds_count{target="%[2]s"} 1
	`, subsystem, egressTarget, fmt.Sprintf("%.16f", randomFloat))

	if err := testutil.CollectAndCompare(
		*m.egressReqDur,
		strings.NewReader(expected),
		fmt.Sprintf("%s_egress_request_duration_seconds", subsystem)); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestRecordIngressRequestCnt(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics(subsystem, true)

	m.RecordIngressRequestCnt(ingressCode, ingressMethod, ingressURL)

	test.Equal(1, testutil.CollectAndCount(*m.ingressReqCnt))
	test.Equal(float64(1), testutil.ToFloat64((*m.ingressReqCnt).WithLabelValues(ingressCode, ingressMethod, ingressURL)))
}

func TestRecordIngressRequestDur(t *testing.T) {
	m := NewMetrics(subsystem, true)

	randomFloat := generateFloatRand(minRand, maxRand)
	m.RecordIngressRequestDur(ingressCode, ingressMethod, ingressURL, randomFloat)

	expected := fmt.Sprintf(`
		# HELP %[1]s_ingress_request_duration_seconds The HTTP request latencies in seconds partitioned by status code, HTTP method and url.
		# TYPE %[1]s_ingress_request_duration_seconds histogram
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="0.005"} 0
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="0.01"} 0
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="0.025"} 0
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="0.05"} 0
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="0.1"} 0
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="0.25"} 0
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="0.5"} 0
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="1"} 0
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="2.5"} 1
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="5"} 1
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="10"} 1
		%[1]s_ingress_request_duration_seconds_bucket{code="%[2]s",method="%[3]s",url="%[4]s",le="+Inf"} 1
		%[1]s_ingress_request_duration_seconds_sum{code="%[2]s",method="%[3]s",url="%[4]s"} %[5]s
		%[1]s_ingress_request_duration_seconds_count{code="%[2]s",method="%[3]s",url="%[4]s"} 1
	`, subsystem, ingressCode, ingressMethod, ingressURL, fmt.Sprintf("%.16f", randomFloat))

	if err := testutil.CollectAndCompare(
		*m.ingressReqDur,
		strings.NewReader(expected),
		fmt.Sprintf("%s_ingress_request_duration_seconds", subsystem)); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}
