package service

import (
	"context"
	"fmt"
	"time"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/pkg/logctx"
	"office-space-allocation/apps/backend/internal/repository"
)

const analyticsReservationLimit = 10_000

// AnalyticsSummary contains dashboard metrics returned by the analytics endpoint.
type AnalyticsSummary struct {
	AverageOccupancy  float64
	PeakDay           string
	PeakOccupancy     float64
	AutoPickRatio     float64
	TotalReservations int
	EarlyReleases     int
	TopZone           AnalyticsTopZone
}

// AnalyticsTopZone describes the most used zone in the selected interval.
type AnalyticsTopZone struct {
	ZoneID string
	Name   string
}

// AnalyticsService provides aggregated dashboard metrics.
type AnalyticsService struct {
	reservations repository.ReservationRepository
	desks        repository.DeskRepository
	zones        repository.ZoneRepository
}

// AnalyticsSummaryInput contains validated analytics filters.
type AnalyticsSummaryInput struct {
	FloorID string
	From    *time.Time
	To      *time.Time
}

// NewAnalyticsService creates an analytics service.
func NewAnalyticsService(reservations repository.ReservationRepository, desks repository.DeskRepository, zones repository.ZoneRepository) *AnalyticsService {
	return &AnalyticsService{
		reservations: reservations,
		desks:        desks,
		zones:        zones,
	}
}

// GetSummary returns a baseline analytics summary derived from reservations and catalog data.
func (service *AnalyticsService) GetSummary(ctx context.Context, input AnalyticsSummaryInput) (AnalyticsSummary, error) {
	if input.From != nil && input.To != nil && !input.From.Before(*input.To) {
		return AnalyticsSummary{}, fmt.Errorf("from must be earlier than to: %w", domain.ErrValidation)
	}

	if (input.From != nil) != (input.To != nil) {
		return AnalyticsSummary{}, fmt.Errorf("from and to must be provided together: %w", domain.ErrValidation)
	}

	reservations, err := service.reservations.List(ctx, repository.ReservationFilter{
		FloorID:  input.FloorID,
		StartsAt: input.From,
		EndsAt:   input.To,
	})
	if err != nil {
		return AnalyticsSummary{}, err
	}

	if len(reservations) >= analyticsReservationLimit {
		logctx.Logger(ctx).Warn("analytics reservation limit reached; results may be incomplete",
			"limit", analyticsReservationLimit,
			"floorID", input.FloorID,
		)
	}

	desks, err := service.desks.List(ctx, repository.DeskFilter{FloorID: input.FloorID})
	if err != nil {
		return AnalyticsSummary{}, err
	}

	zoneNames := map[string]string{}
	var zoneLoadErr error
	var loadedZones []domain.Zone
	if input.FloorID != "" {
		loadedZones, zoneLoadErr = service.zones.ListByFloorID(ctx, input.FloorID)
	} else {
		loadedZones, zoneLoadErr = service.zones.List(ctx)
	}
	if zoneLoadErr != nil {
		return AnalyticsSummary{}, zoneLoadErr
	}
	for _, zone := range loadedZones {
		zoneNames[zone.ID] = zone.Name
	}

	deskByID := make(map[string]domain.Desk, len(desks))
	for _, desk := range desks {
		deskByID[desk.ID] = desk
	}

	totalReservations := len(reservations)
	autoReservations := 0
	earlyReleases := 0
	zoneUsage := map[string]int{}
	dayUsage := map[string]int{}

	for _, reservation := range reservations {
		if reservation.Source == domain.ReservationSourceAuto {
			autoReservations++
		}
		if reservation.ReleasedAt != nil {
			earlyReleases++
		}

		dayKey := reservation.StartsAt.UTC().Weekday().String()
		dayUsage[dayKey]++

		desk, ok := deskByID[reservation.DeskID]
		if !ok {
			continue
		}
		if desk.ZoneID != "" {
			zoneUsage[desk.ZoneID]++
		}
	}

	averageOccupancy := 0.0
	if len(desks) > 0 {
		if input.From != nil && input.To != nil {
			// Time-weighted occupancy: total non-cancelled desk-seconds / (desks × period seconds).
			periodSeconds := input.To.Sub(*input.From).Seconds()
			if periodSeconds > 0 {
				var occupiedSeconds float64
				for _, reservation := range reservations {
					if reservation.Status == domain.ReservationStatusCancelled {
						continue
					}
					start := reservation.StartsAt
					if start.Before(*input.From) {
						start = *input.From
					}
					end := reservation.EndsAt
					if end.After(*input.To) {
						end = *input.To
					}
					if end.After(start) {
						occupiedSeconds += end.Sub(start).Seconds()
					}
				}
				averageOccupancy = occupiedSeconds / (float64(len(desks)) * periodSeconds)
			}
		} else {
			// No time range: fraction of desks with at least one active reservation.
			activeCount := 0
			for _, r := range reservations {
				if r.Status == domain.ReservationStatusActive {
					activeCount++
				}
			}
			averageOccupancy = float64(activeCount) / float64(len(desks))
		}
	}

	peakDay := ""
	peakOccupancy := 0.0
	peakReservations := 0
	for day, count := range dayUsage {
		if count > peakReservations {
			peakReservations = count
			peakDay = day
		}
	}
	if len(desks) > 0 {
		peakOccupancy = float64(peakReservations) / float64(len(desks))
	}

	autoPickRatio := 0.0
	if totalReservations > 0 {
		autoPickRatio = float64(autoReservations) / float64(totalReservations)
	}

	topZoneID := ""
	topZoneCount := 0
	for zoneID, count := range zoneUsage {
		if count > topZoneCount {
			topZoneID = zoneID
			topZoneCount = count
		}
	}

	return AnalyticsSummary{
		AverageOccupancy:  averageOccupancy,
		PeakDay:           peakDay,
		PeakOccupancy:     peakOccupancy,
		AutoPickRatio:     autoPickRatio,
		TotalReservations: totalReservations,
		EarlyReleases:     earlyReleases,
		TopZone: AnalyticsTopZone{
			ZoneID: topZoneID,
			Name:   zoneNames[topZoneID],
		},
	}, nil
}
