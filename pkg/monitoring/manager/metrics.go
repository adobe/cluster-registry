package monitoring

import "github.com/prometheus/client_golang/prometheus"
import "github.com/prometheus/client_golang/prometheus/promauto"

type MetricsI interface {
	RecordRequeueCnt(target string)
	RecordReconciliationCnt(target string)
	RecordReconciliationDur(target string, elapsed float64)
	RecordEnqueueCnt(target string)
	RecordEnqueueDur(target string, elapsed float64)
	RecordErrorCnt(target string)
}

type Metrics struct {
	RequeueCnt        *prometheus.CounterVec
	ReconciliationCnt *prometheus.CounterVec
	ReconciliationDur *prometheus.HistogramVec
	EnqueueCnt        *prometheus.CounterVec
	EnqueueDur        *prometheus.HistogramVec
	ErrCnt            *prometheus.CounterVec
	metrics           []prometheus.Collector
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) Init(isUnitTest bool) {
	reg := prometheus.DefaultRegisterer
	if isUnitTest {
		reg = prometheus.NewRegistry()
	}
	var requeueCnt prometheus.Collector = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Name: "cluster_registry_sync_manager_requeues_total",
		Help: "The total number of controller-manager requeues partitioned by target.",
	}, []string{"target"})
	m.RequeueCnt = requeueCnt.(*prometheus.CounterVec)
	m.metrics = append(m.metrics, m.RequeueCnt)

	var reconciliationCnt prometheus.Collector = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Name: "cluster_registry_sync_manager_reconciliation_total",
		Help: "How many reconciliations occurred, partitioned by target.",
	},
		[]string{"target"},
	)
	m.ReconciliationCnt = reconciliationCnt.(*prometheus.CounterVec)
	m.metrics = append(m.metrics, m.ReconciliationCnt)

	var reconciliationDur prometheus.Collector = promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
		Name: "cluster_registry_sync_manager_reconciliation_duration_seconds",
		Help: "The time taken to reconcile resources in seconds partitioned by target.",
	},
		[]string{"target"},
	)
	m.ReconciliationDur = reconciliationDur.(*prometheus.HistogramVec)
	m.metrics = append(m.metrics, m.ReconciliationDur)

	var enqueueCnt prometheus.Collector = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Name: "cluster_registry_sync_manager_enqueue_total",
		Help: "How many reconciliations were enqueued, partitioned by target.",
	},
		[]string{"target"},
	)
	m.EnqueueCnt = enqueueCnt.(*prometheus.CounterVec)
	m.metrics = append(m.metrics, m.EnqueueCnt)

	var enqueueDur prometheus.Collector = promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
		Name: "cluster_registry_sync_manager_enqueue_duration_seconds",
		Help: "The time taken to enqueue a reconciliation in seconds partitioned by target.",
	},
		[]string{"target"},
	)
	m.EnqueueDur = enqueueDur.(*prometheus.HistogramVec)
	m.metrics = append(m.metrics, m.EnqueueDur)

	var errorCnt prometheus.Collector = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Name: "cluster_registry_sync_manager_error_total",
		Help: "The total number controller-manager errors partitioned by target.",
	}, []string{"target"})
	m.ErrCnt = errorCnt.(*prometheus.CounterVec)
	m.metrics = append(m.metrics, m.ErrCnt)
}

func (m *Metrics) RecordRequeueCnt(target string) {
	m.RequeueCnt.WithLabelValues(target).Inc()
}

func (m *Metrics) RecordReconciliationCnt(target string) {
	m.ReconciliationCnt.WithLabelValues(target).Inc()
}

func (m *Metrics) RecordReconciliationDur(target string, elapsed float64) {
	m.ReconciliationDur.WithLabelValues(target).Observe(elapsed)
}

func (m *Metrics) RecordEnqueueCnt(target string) {
	m.EnqueueCnt.WithLabelValues(target).Inc()
}

func (m *Metrics) RecordEnqueueDur(target string, elapsed float64) {
	m.EnqueueDur.WithLabelValues(target).Observe(elapsed)
}

func (m *Metrics) RecordErrorCnt(target string) {
	m.ErrCnt.WithLabelValues(target).Inc()
}
