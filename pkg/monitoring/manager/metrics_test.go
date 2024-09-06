package monitoring

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"strings"
	"testing"
)

const (
	clusterSyncTarget = "orgnumber-env-region-cluster-sync"
	subsystem         = "cluster_registry_sync_manager"
	minRand           = 1
	maxRand           = 2.5
)

// Generate a random float number between min and max
func generateFloatRand(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// Generate what we expect a histogram of some random number to look like. metricTopic is what the metric is about, e.g.
// reconciliation or enqueue. helpString is the literal help string from metrics.go. I'd grab this myself, but it's not
// exposed in the HistogramVec object AFAICT :(
func generateExpectedHistogram(randomFloat float64, metricTopic string, helpString string) string {
	expected := fmt.Sprintf(`
		# HELP %[1]s_%[5]s_duration_seconds %[4]s
		# TYPE %[1]s_%[5]s_duration_seconds histogram
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="0.005"} 0
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="0.01"} 0
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="0.025"} 0
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="0.05"} 0
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="0.1"} 0
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="0.25"} 0
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="0.5"} 0
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="1"} 0
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="2.5"} 1
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="5"} 1
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="10"} 1
		%[1]s_%[5]s_duration_seconds_bucket{target="%[2]s",le="+Inf"} 1
		%[1]s_%[5]s_duration_seconds_sum{target="%[2]s"} %[3]s
		%[1]s_%[5]s_duration_seconds_count{target="%[2]s"} 1
	`, subsystem, clusterSyncTarget, fmt.Sprintf("%.16f", randomFloat), helpString, metricTopic)
	return expected
}

func TestNewMetrics(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics()
	test.NotNil(m)
}

func TestInit(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics()
	m.Init(true)
	test.NotNil(m.RequeueCnt)
	test.NotNil(m.ReconciliationCnt)
	test.NotNil(m.ReconciliationDur)
	test.NotNil(m.EnqueueCnt)
	test.NotNil(m.EnqueueDur)
	test.NotNil(m.ErrCnt)
}

func TestRecordRequeueCnt(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics()
	m.Init(true)
	m.RecordRequeueCnt(clusterSyncTarget)
	test.Equal(1, testutil.CollectAndCount(*m.RequeueCnt))
	test.Equal(float64(1), testutil.ToFloat64((*m.RequeueCnt).WithLabelValues(clusterSyncTarget)))
}

func TestRecordReconciliationCnt(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics()
	m.Init(true)
	m.RecordReconciliationCnt(clusterSyncTarget)
	test.Equal(1, testutil.CollectAndCount(*m.ReconciliationCnt))
	test.Equal(float64(1), testutil.ToFloat64((*m.ReconciliationCnt).WithLabelValues(clusterSyncTarget)))
}

func TestRecordReconciliationDur(t *testing.T) {
	m := NewMetrics()
	m.Init(true)
	randomFloat := generateFloatRand(minRand, maxRand)
	m.RecordReconciliationDur(clusterSyncTarget, randomFloat)
	expected := generateExpectedHistogram(randomFloat, "reconciliation", "The time taken to reconcile resources in seconds partitioned by target.")
	if err := testutil.CollectAndCompare(
		*m.ReconciliationDur,
		strings.NewReader(expected),
		fmt.Sprintf("%s_%s_duration_seconds", subsystem, "reconciliation")); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}

}

func TestRecordEnqueueCnt(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics()
	m.Init(true)
	m.RecordEnqueueCnt(clusterSyncTarget)
	test.Equal(1, testutil.CollectAndCount(*m.EnqueueCnt))
	test.Equal(float64(1), testutil.ToFloat64((*m.EnqueueCnt).WithLabelValues(clusterSyncTarget)))

}

func TestRecordEnqueueDur(t *testing.T) {
	m := NewMetrics()
	m.Init(true)
	randomFloat := generateFloatRand(minRand, maxRand)
	m.RecordEnqueueDur(clusterSyncTarget, randomFloat)
	expected := generateExpectedHistogram(randomFloat, "enqueue", "The time taken to enqueue a reconciliation in seconds partitioned by target.")
	if err := testutil.CollectAndCompare(
		*m.EnqueueDur,
		strings.NewReader(expected),
		fmt.Sprintf("%s_%s_duration_seconds", subsystem, "enqueue")); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}

}

func TestRecordErrCnt(t *testing.T) {
	test := assert.New(t)
	m := NewMetrics()
	m.Init(true)
	m.RecordErrorCnt(clusterSyncTarget)
	test.Equal(1, testutil.CollectAndCount(*m.ErrCnt))
	test.Equal(float64(1), testutil.ToFloat64((*m.ErrCnt).WithLabelValues(clusterSyncTarget)))

}
