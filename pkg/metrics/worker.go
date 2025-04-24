package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// WorkerPoolGauges tracks worker pool gauges
	WorkerPoolGauges = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "worker_pool_gauges",
			Help: "Worker pool gauges by pool name and type",
		},
		[]string{"pool", "type"},
	)

	// WorkerPoolCounters tracks worker pool counters
	WorkerPoolCounters = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_pool_counters",
			Help: "Worker pool counters by pool name and type",
		},
		[]string{"pool", "type"},
	)

	// WorkerPoolHistograms tracks worker pool timing metrics
	WorkerPoolHistograms = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "worker_pool_processing_seconds",
			Help:    "Worker pool processing time in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"pool"},
	)
)
