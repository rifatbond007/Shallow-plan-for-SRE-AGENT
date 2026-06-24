package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sre_agent_http_requests_total",
		Help: "Total HTTP requests by method, path, status",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sre_agent_http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	ClaudeRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sre_agent_claude_requests_total",
		Help: "Total Claude API calls",
	})

	ClaudeRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "sre_agent_claude_request_duration_seconds",
		Help:    "Claude API call latency",
		Buckets: prometheus.DefBuckets,
	})

	HypothesesGenerated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sre_agent_hypotheses_generated_total",
		Help: "Total hypotheses generated",
	})

	CacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sre_agent_cache_hits_total",
		Help: "Total cache hits",
	})

	CacheMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sre_agent_cache_misses_total",
		Help: "Total cache misses",
	})
)
