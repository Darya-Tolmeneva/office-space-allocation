package service

import (
	"context"
	"fmt"
	"time"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/repository"
)

// CatalogService provides read operations for floors, zones, and desks.
type CatalogService struct {
	floors repository.FloorRepository
	zones  repository.ZoneRepository
	desks  repository.DeskRepository
}

// ListDesksInput contains validated desk list filters.
type ListDesksInput struct {
	FloorID  string
	ZoneID   string
	Features []domain.DeskFeature
}

// DeskAvailabilitySlot contains desk availability data for API responses.
type DeskAvailabilitySlot struct {
	From          time.Time
	To            time.Time
	Status        string
	ReservationID string
}

// FloorDetails contains a floor with related zones and desks.
type FloorDetails struct {
	Floor domain.Floor
	Zones []domain.Zone
	Desks []domain.Desk
}

// NewCatalogService creates a catalog service.
func NewCatalogService(floors repository.FloorRepository, zones repository.ZoneRepository, desks repository.DeskRepository) *CatalogService {
	return &CatalogService{
		floors: floors,
		zones:  zones,
		desks:  desks,
	}
}

// ListFloors returns all floors.
func (service *CatalogService) ListFloors(ctx context.Context) ([]domain.Floor, error) {
	return service.floors.List(ctx)
}

// GetFloorDetails returns a floor with its zones and desks.
func (service *CatalogService) GetFloorDetails(ctx context.Context, floorID string) (FloorDetails, error) {
	floor, err := service.floors.GetByID(ctx, floorID)
	if err != nil {
		return FloorDetails{}, err
	}

	zones, err := service.zones.ListByFloorID(ctx, floorID)
	if err != nil {
		return FloorDetails{}, err
	}

	desks, err := service.desks.List(ctx, repository.DeskFilter{FloorID: floorID})
	if err != nil {
		return FloorDetails{}, err
	}

	return FloorDetails{
		Floor: floor,
		Zones: zones,
		Desks: desks,
	}, nil
}

// ListDesks returns desks filtered by validated input.
func (service *CatalogService) ListDesks(ctx context.Context, input ListDesksInput) ([]domain.Desk, error) {
	return service.desks.List(ctx, repository.DeskFilter{
		FloorID:  input.FloorID,
		ZoneID:   input.ZoneID,
		Features: input.Features,
	})
}

// GetDesk returns a desk by identifier.
func (service *CatalogService) GetDesk(ctx context.Context, deskID string) (domain.Desk, error) {
	return service.desks.GetByID(ctx, deskID)
}

// GetDeskAvailability returns availability slots for a desk within the requested interval.
func (service *CatalogService) GetDeskAvailability(ctx context.Context, deskID string, startsAt time.Time, endsAt time.Time) ([]DeskAvailabilitySlot, error) {
	if startsAt.IsZero() || endsAt.IsZero() {
		return nil, fmt.Errorf("from and to are required: %w", domain.ErrValidation)
	}
	if !startsAt.Before(endsAt) {
		return nil, fmt.Errorf("from must be earlier than to: %w", domain.ErrValidation)
	}

	slots, err := service.desks.ListAvailability(ctx, deskID, startsAt, endsAt)
	if err != nil {
		return nil, err
	}

	result := make([]DeskAvailabilitySlot, 0, len(slots))
	for _, slot := range slots {
		result = append(result, DeskAvailabilitySlot{
			From:          slot.From,
			To:            slot.To,
			Status:        slot.Status,
			ReservationID: slot.ReservationID,
		})
	}

	return result, nil
}
