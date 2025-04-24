package metrics

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SystemGauges tracks system metrics
	SystemGauges = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_stats",
			Help: "System statistics",
		},
		[]string{"type"},
	)

	// GCStats tracks garbage collection metrics
	GCStats = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gc_stats",
			Help: "Garbage collection statistics",
		},
		[]string{"type"},
	)

	// HeapStats tracks heap memory metrics
	HeapStats = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "heap_stats",
			Help: "Heap memory statistics",
		},
		[]string{"type"},
	)
)

// CollectSystemMetrics periodically collects system metrics
func CollectSystemMetrics(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)

			// System stats
			SystemGauges.WithLabelValues("goroutines").Set(float64(runtime.NumGoroutine()))
			SystemGauges.WithLabelValues("cgo_calls").Set(float64(runtime.NumCgoCall()))
			SystemGauges.WithLabelValues("cpu_threads").Set(float64(runtime.GOMAXPROCS(0)))

			// GC stats
			GCStats.WithLabelValues("num_gc").Set(float64(stats.NumGC))
			GCStats.WithLabelValues("pause_total_ns").Set(float64(stats.PauseTotalNs))
			GCStats.WithLabelValues("last_pause_ns").Set(float64(stats.PauseNs[(stats.NumGC+255)%256]))

			// Heap stats
			HeapStats.WithLabelValues("alloc").Set(float64(stats.HeapAlloc))
			HeapStats.WithLabelValues("sys").Set(float64(stats.HeapSys))
			HeapStats.WithLabelValues("idle").Set(float64(stats.HeapIdle))
			HeapStats.WithLabelValues("inuse").Set(float64(stats.HeapInuse))
			HeapStats.WithLabelValues("released").Set(float64(stats.HeapReleased))
			HeapStats.WithLabelValues("objects").Set(float64(stats.HeapObjects))
		}
	}()
}
