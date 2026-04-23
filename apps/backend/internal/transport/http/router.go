package http

import (
	"context"
	"log/slog"
	stdhttp "net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"office-space-allocation/apps/backend/internal/pkg/config"
	"office-space-allocation/apps/backend/internal/pkg/logctx"
	"office-space-allocation/apps/backend/internal/pkg/metrics"
)

const maxRequestBodyBytes = 1 << 20 // 1 MB

// Dependencies contains transport dependencies used to build the router.
type Dependencies struct {
	AuthService        authService
	WorkspaceService   workspaceService
	ReservationService reservationService
	AnalyticsService   analyticsService
	AuthConfig         config.AuthConfig
	AllowedOrigins     []string
	DBPing             func(context.Context) error
}

// NewRouter builds the baseline HTTP router.
func NewRouter(dependencies Dependencies) stdhttp.Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(metrics.Middleware)
	router.Use(slogMiddleware(logger))
	router.Use(maxBodySizeMiddleware(maxRequestBodyBytes))
	router.Use(corsMiddleware(dependencies.AllowedOrigins))

	router.Get("/healthz", handleHealth(dependencies.DBPing))
	router.Handle("/metrics", promhttp.Handler())

	router.Route("/v1", func(router chi.Router) {
		if dependencies.AuthService != nil {
			authHandler := NewAuthHandler(dependencies.AuthService)
			router.Route("/auth", func(router chi.Router) {
				router.Post("/register", authHandler.Register)
				router.Post("/login", authHandler.Login)
				router.Post("/refresh", authHandler.Refresh)
			})
		}

		if dependencies.WorkspaceService != nil {
			floorHandler := NewFloorHandler(dependencies.WorkspaceService)
			deskHandler := NewDeskHandler(dependencies.WorkspaceService)
			router.Get("/floors", floorHandler.List)
			router.Get("/floors/{floorId}", floorHandler.Get)
			router.Get("/desks", deskHandler.List)
			router.Get("/desks/{deskId}", deskHandler.Get)
			router.Get("/desks/{deskId}/availability", deskHandler.Availability)
		}

		if dependencies.ReservationService != nil {
			reservationHandler := NewReservationHandler(dependencies.ReservationService)
			requireAuth := RequireAuth(dependencies.AuthConfig.JWTSigningKey)
			router.With(requireAuth).Get("/reservations", reservationHandler.List)
			router.With(requireAuth).Post("/reservations", reservationHandler.Create)
			router.With(requireAuth).Post("/reservations/auto", reservationHandler.CreateAuto)
			router.With(requireAuth).Post("/reservations/auto/preview", reservationHandler.PreviewAuto)
			router.With(requireAuth).Get("/reservations/{reservationId}", reservationHandler.Get)
			router.With(requireAuth).Delete("/reservations/{reservationId}", reservationHandler.Cancel)
			router.With(requireAuth).Post("/reservations/{reservationId}/release", reservationHandler.Release)
		}

		if dependencies.AnalyticsService != nil {
			analyticsHandler := NewAnalyticsHandler(dependencies.AnalyticsService)
			router.Get("/analytics/summary", analyticsHandler.Summary)
		}
	})

	return router
}

// handleHealth serves a readiness probe at GET /healthz.
// Returns 503 if the database is unreachable.
func handleHealth(ping func(context.Context) error) stdhttp.HandlerFunc {
	return func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		if ping != nil {
			if err := ping(request.Context()); err != nil {
				writeJSON(writer, stdhttp.StatusServiceUnavailable, map[string]string{"status": "unhealthy"})
				return
			}
		}
		writeJSON(writer, stdhttp.StatusOK, map[string]string{"status": "ok"})
	}
}

// maxBodySizeMiddleware limits the size of incoming request bodies.
func maxBodySizeMiddleware(limit int64) func(stdhttp.Handler) stdhttp.Handler {
	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			r.Body = stdhttp.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware returns a CORS handler that restricts origins to the provided list.
// If the list is empty, any origin is allowed (suitable for local development).
// In production, set ALLOWED_ORIGINS to a comma-separated list of permitted origins.
func corsMiddleware(allowedOrigins []string) func(stdhttp.Handler) stdhttp.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			_, originAllowed := allowed[origin]
			if len(allowed) > 0 && !originAllowed {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == stdhttp.MethodOptions {
				w.WriteHeader(stdhttp.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// slogMiddleware logs each request and stores a request-scoped logger with requestId in context.
func slogMiddleware(logger *slog.Logger) func(stdhttp.Handler) stdhttp.Handler {
	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			requestLogger := logger.With("requestId", middleware.GetReqID(r.Context()))
			r = r.WithContext(logctx.WithLogger(r.Context(), requestLogger))
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			requestLogger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
			)
		})
	}
}
