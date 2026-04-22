package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"

	"office-space-allocation/apps/backend/internal/job"
	"office-space-allocation/apps/backend/internal/pkg/config"
	"office-space-allocation/apps/backend/internal/repository/postgres"
	"office-space-allocation/apps/backend/internal/service"
	httptransport "office-space-allocation/apps/backend/internal/transport/http"
)

// App wires infrastructure and transport for the backend service.
type App struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
	database        *sqlx.DB
	expiryJob       *job.ExpiryJob
}

// New builds the application with baseline dependencies.
func New() (*App, error) {
	applicationConfig, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	database, err := postgres.Open(context.Background(), postgres.Config{
		DSN:             applicationConfig.Postgres.DSN,
		MaxOpenConns:    applicationConfig.Postgres.MaxOpenConns,
		MaxIdleConns:    applicationConfig.Postgres.MaxIdleConns,
		ConnMaxLifetime: applicationConfig.Postgres.ConnMaxLifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	userRepository := postgres.NewUserRepository(database)
	refreshTokenRepository := postgres.NewRefreshTokenRepository(database)
	floorRepository := postgres.NewFloorRepository(database)
	zoneRepository := postgres.NewZoneRepository(database)
	deskRepository := postgres.NewDeskRepository(database)
	reservationRepository := postgres.NewReservationRepository(database)

	authService := service.NewAuthService(userRepository, refreshTokenRepository, applicationConfig.Auth)
	workspaceService := service.NewCatalogService(
		floorRepository,
		zoneRepository,
		deskRepository,
	)
	reservationService := service.NewReservationService(
		reservationRepository,
		deskRepository,
		userRepository,
	)
	analyticsService := service.NewAnalyticsService(
		reservationRepository,
		deskRepository,
		zoneRepository,
	)

	handler := httptransport.NewRouter(httptransport.Dependencies{
		AuthService:        authService,
		WorkspaceService:   workspaceService,
		ReservationService: reservationService,
		AnalyticsService:   analyticsService,
		AuthConfig:         applicationConfig.Auth,
		AllowedOrigins:     applicationConfig.HTTP.AllowedOrigins,
		DBPing:             database.PingContext,
	})

	server := &http.Server{
		Addr:              applicationConfig.HTTP.Address,
		Handler:           handler,
		ReadHeaderTimeout: applicationConfig.HTTP.ReadHeaderTimeout,
		WriteTimeout:      applicationConfig.HTTP.WriteTimeout,
	}

	return &App{
		httpServer:      server,
		shutdownTimeout: applicationConfig.HTTP.ShutdownTimeout,
		database:        database,
		expiryJob:       job.NewExpiryJob(reservationRepository, time.Minute),
	}, nil
}

// Run starts the application and shuts it down gracefully on context cancellation.
func (application *App) Run(ctx context.Context) error {
	defer func() {
		_ = application.database.Close()
	}()

	slog.Info("server listening", "address", application.httpServer.Addr)

	errorChannel := make(chan error, 1)

	go application.expiryJob.Run(ctx)

	go func() {
		if err := application.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorChannel <- fmt.Errorf("listen and serve: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), application.shutdownTimeout)
		defer cancel()

		if err := application.httpServer.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}

		return nil
	case err := <-errorChannel:
		return err
	}
}
