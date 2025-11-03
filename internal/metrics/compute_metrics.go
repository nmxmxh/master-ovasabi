package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CapabilityUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "compute_capability_updates_total",
		Help: "Total number of compute capability update events received (from clients)",
	})

	ComputeAssigned = promauto.NewCounter(prometheus.CounterOpts{
		Name: "compute_assigned_total",
		Help: "Total number of compute:dispatch:v1:assigned events successfully delivered to a client",
	})

	ComputeAssignmentFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "compute_assignment_failures_total",
		Help: "Total number of compute assignments that could not be delivered (no bound client)",
	})

	DedupDrops = promauto.NewCounter(prometheus.CounterOpts{
		Name: "event_dedup_drops_total",
		Help: "Total number of events dropped due to deduplication logic",
	})

	PendingRequestsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pending_requests",
		Help: "Current size of the pending request map for request/response matching",
	})
)

// Init is a no-op now because promauto registers on var init, but keeping for explicit calls.
func Init() {
	// ensure variables are referenced so linters are happy in builds without Prometheus
	_ = CapabilityUpdates
	_ = ComputeAssigned
	_ = ComputeAssignmentFailures
	_ = DedupDrops
	_ = PendingRequestsGauge
}

// Helper functions
func IncCapabilityUpdates()    { CapabilityUpdates.Inc() }
func IncComputeAssigned()      { ComputeAssigned.Inc() }
func IncAssignmentFailures()   { ComputeAssignmentFailures.Inc() }
func IncDedupDrops()           { DedupDrops.Inc() }
func SetPendingRequests(n int) { PendingRequestsGauge.Set(float64(n)) }
