// Package metrics defines Prometheus metrics for the API + worker
// and provides middleware to instrument HTTP handlers.
//
// Two consumers of this package:
//   - cmd/api  → uses Middleware() to wrap router; exposes /metrics handler
//   - cmd/worker → uses MessageProcessed/MessageFailed counters
//
// Conventions:
//   - All metrics prefixed with "studentsystemgo_" (project namespace)
//   - Labels are LOW-CARDINALITY (status code, method, route name) — never user IDs
//     Reason: each label combination = a separate time-series. High-cardinality
//     labels (e.g., user_id) explode storage and crash Prometheus.
package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ── HTTP metrics ──────────────────────────────────────────

// HTTPRequestsTotal counts every HTTP request handled.
// Labels: method (GET/POST/...), path (route pattern), status (200, 4xx, 5xx).
//
// Why path PATTERN not exact URL? `/students/42` and `/students/43` both
// get labeled `/students/{id}`. Otherwise we'd create one time-series per
// student ID — high cardinality, breaks Prometheus.
var HTTPRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "studentsystemgo_http_requests_total",
		Help: "Total HTTP requests by method, path, and status code.",
	},
	[]string{"method", "path", "status"},
)

// HTTPRequestDuration is a histogram of request latencies in seconds.
// Buckets chosen for typical web API latency: 1ms to 5s.
//
// Default Prometheus buckets are tuned for slow services (5ms-10s);
// our app should be sub-100ms most of the time, so we use finer buckets.
var HTTPRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "studentsystemgo_http_request_duration_seconds",
		Help:    "HTTP request duration by method and path.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	},
	[]string{"method", "path"},
)

// ── Worker metrics ────────────────────────────────────────

// MessagesProcessed counts messages successfully consumed and ACKed.
// Labels: queue (which queue the worker pulled from).
var MessagesProcessed = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "studentsystemgo_worker_messages_processed_total",
		Help: "Worker messages processed successfully (ACKed).",
	},
	[]string{"queue"},
)

// MessagesFailed counts messages that the worker rejected (NACKed).
// These end up in the failed queue (DLX). Useful for alerting.
var MessagesFailed = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "studentsystemgo_worker_messages_failed_total",
		Help: "Worker messages NACKed (sent to DLX).",
	},
	[]string{"queue"},
)

// ── HTTP middleware ───────────────────────────────────────

// Middleware wraps a handler to record request metrics.
//
// Self-exclusions:
//   - /metrics: don't record scrapes (creates a feedback loop in dashboards)
//
// Path label normalization:
//   - Use r.Pattern when available (gives "/students/{id}" not "/students/42")
//   - Strip leading "GET " etc. (Go's mux includes the method in Pattern)
//   - Fall back to r.URL.Path when no pattern matched (e.g., 401/404 paths)
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip the scrape endpoint itself.
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		duration := time.Since(start).Seconds()

		path := normalizePath(r)
		labels := prometheus.Labels{
			"method": r.Method,
			"path":   path,
			"status": strconv.Itoa(rec.status),
		}
		HTTPRequestsTotal.With(labels).Inc()
		HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// normalizePath returns a low-cardinality path label.
// Go 1.22's mux puts "METHOD /path/{var}" in r.Pattern; we strip the method.
func normalizePath(r *http.Request) string {
	p := r.Pattern
	if p == "" {
		return r.URL.Path // unmatched route — already low-cardinality (401/404 paths)
	}
	// Strip method prefix: "GET /healthz" -> "/healthz"
	if i := strings.Index(p, " "); i >= 0 {
		return p[i+1:]
	}
	return p
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
// Without wrapping, there's no way to know what status the handler wrote.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
