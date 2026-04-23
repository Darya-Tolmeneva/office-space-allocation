package metrics

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	stdhttp "net/http"
)

var (
	// httpRequestsTotal counts HTTP requests by method, path pattern, and status code
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "flowdesk",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	// httpRequestDuration tracks request latency in seconds
	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "flowdesk",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"method", "path", "status"})

	// httpResponseSize tracks response body size in bytes
	httpResponseSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "flowdesk",
		Subsystem: "http",
		Name:      "response_size_bytes",
		Help:      "HTTP response body size in bytes.",
		Buckets:   prometheus.ExponentialBuckets(100, 10, 6), // 100, 1K, 10K, 100K, 1M, 10M
	}, []string{"method", "path", "status"})

	// httpRequestsInFlight tracks the number of in-flight HTTP requests
	httpRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "flowdesk",
		Subsystem: "http",
		Name:      "requests_in_flight",
		Help:      "Number of HTTP requests currently being processed.",
	})
)

var (
	// ActiveReservations is a gauge for the number of currently active reservations
	ActiveReservations = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "flowdesk",
		Subsystem: "business",
		Name:      "active_reservations",
		Help:      "Number of currently active reservations.",
	})

	// TotalDesks is a gauge for the total number of desks in the system
	TotalDesks = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "flowdesk",
		Subsystem: "business",
		Name:      "total_desks",
		Help:      "Total number of desks.",
	})

	// TotalFloors is a gauge for the total number of floors
	TotalFloors = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "flowdesk",
		Subsystem: "business",
		Name:      "total_floors",
		Help:      "Total number of floors.",
	})

	// ReservationsCreatedTotal counts reservations created since process start
	ReservationsCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "flowdesk",
		Subsystem: "business",
		Name:      "reservations_created_total",
		Help:      "Total number of reservations created, by source (manual/auto).",
	}, []string{"source"})

	// ReservationsCancelledTotal counts reservations cancelled since process start
	ReservationsCancelledTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "flowdesk",
		Subsystem: "business",
		Name:      "reservations_cancelled_total",
		Help:      "Total number of reservations cancelled.",
	})

	// ReservationsReleasedTotal counts reservations released early since process start
	ReservationsReleasedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "flowdesk",
		Subsystem: "business",
		Name:      "reservations_released_total",
		Help:      "Total number of reservations released early.",
	})

	// AuthRegistrationsTotal counts user registrations since process start
	AuthRegistrationsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "flowdesk",
		Subsystem: "auth",
		Name:      "registrations_total",
		Help:      "Total number of user registrations.",
	})

	// AuthLoginsTotal counts login attempts by result
	AuthLoginsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "flowdesk",
		Subsystem: "auth",
		Name:      "logins_total",
		Help:      "Total number of login attempts.",
	}, []string{"result"})

	// DBQueryDuration tracks database query latency
	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "flowdesk",
		Subsystem: "db",
		Name:      "query_duration_seconds",
		Help:      "Database query duration in seconds.",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 5},
	}, []string{"operation"})
)

// Middleware returns a chi-compatible middleware that records Prometheus metrics
func Middleware(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		start := time.Now()
		httpRequestsInFlight.Inc()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		httpRequestsInFlight.Dec()

		status := strconv.Itoa(ww.Status())
		pattern := routePattern(r)
		method := r.Method
		elapsed := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(method, pattern, status).Inc()
		httpRequestDuration.WithLabelValues(method, pattern, status).Observe(elapsed)
		httpResponseSize.WithLabelValues(method, pattern, status).Observe(float64(ww.BytesWritten()))
	})
}

// routePattern extracts the chi route pattern for the current request
func routePattern(r *stdhttp.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return sanitizePath(r.URL.Path)
	}

	pattern := rctx.RoutePattern()
	if pattern == "" {
		return sanitizePath(r.URL.Path)
	}

	return pattern
}

// sanitizePath normalises a raw URL path for use as a Prometheus label
func sanitizePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if looksLikeID(part) {
			parts[i] = "{id}"
		}
	}
	return strings.Join(parts, "/")
}

// looksLikeID returns true if the string looks like a UUID or numeric ID
func looksLikeID(s string) bool {
	if s == "" {
		return false
	}
	// UUID: 8-4-4-4-12 hex chars
	if len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-' {
		return true
	}
	// Numeric ID
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
