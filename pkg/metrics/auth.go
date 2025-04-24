package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// AuthRequests tracks the total number of auth requests by type and status
	AuthRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_requests_total",
			Help: "Total number of auth requests by type and status",
		},
		[]string{"type", "status"},
	)

	// AuthLatency tracks the latency of auth operations
	AuthLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_latency_seconds",
			Help:    "Latency of auth operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// ActiveSessions tracks the number of active user sessions
	ActiveSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "auth_active_sessions",
			Help: "Number of active user sessions",
		},
	)

	// TokenErrors tracks JWT token validation errors
	TokenErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_token_errors_total",
			Help: "Total number of token validation errors by type",
		},
		[]string{"error_type"},
	)
)
