package metrics

import (
	"context"
	"log/slog"
	"time"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/repository"
)

// CollectorDeps holds the repositories needed to collect business metrics
type CollectorDeps struct {
	Reservations repository.ReservationRepository
	Desks        repository.DeskRepository
	Floors       repository.FloorRepository
}

// RunCollector starts a background goroutine that periodically refreshes
// business-level Prometheus gauges
func RunCollector(ctx context.Context, deps CollectorDeps, interval time.Duration) {
	// Collect once immediately at startup.
	collect(ctx, deps)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			collect(ctx, deps)
		}
	}
}

func collect(ctx context.Context, deps CollectorDeps) {
	// Active reservations
	if deps.Reservations != nil {
		active, err := deps.Reservations.List(ctx, repository.ReservationFilter{
			Status: domain.ReservationStatusActive,
		})
		if err != nil {
			slog.Warn("metrics collector: failed to count active reservations", "error", err)
		} else {
			ActiveReservations.Set(float64(len(active)))
		}
	}

	// Total desks
	if deps.Desks != nil {
		desks, err := deps.Desks.List(ctx, repository.DeskFilter{})
		if err != nil {
			slog.Warn("metrics collector: failed to count desks", "error", err)
		} else {
			TotalDesks.Set(float64(len(desks)))
		}
	}

	// Total floors
	if deps.Floors != nil {
		floors, err := deps.Floors.List(ctx)
		if err != nil {
			slog.Warn("metrics collector: failed to count floors", "error", err)
		} else {
			TotalFloors.Set(float64(len(floors)))
		}
	}
}
