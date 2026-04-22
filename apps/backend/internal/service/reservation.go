package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/repository"
)

// ReservationDetails contains reservation data enriched with desk metadata required by the API.
type ReservationDetails struct {
	Reservation domain.Reservation
	DeskLabel   string
	FloorID     string
}

// AutoReservationPreview contains the selected desk for auto-pick preview responses.
type AutoReservationPreview struct {
	Desk    domain.Desk
	Score   float64
	Reasons []string
}

// ReservationService handles reservation use cases.
type ReservationService struct {
	reservations repository.ReservationRepository
	desks        repository.DeskRepository
	users        repository.UserRepository
	now          func() time.Time
}

// CreateReservationInput contains validated data for manual reservation creation.
type CreateReservationInput struct {
	DeskID   string
	UserID   string
	Note     string
	StartsAt time.Time
	EndsAt   time.Time
}

// AutoReservationInput contains validated data for baseline auto-pick reservation creation.
type AutoReservationInput struct {
	UserID           string
	StartsAt         time.Time
	EndsAt           time.Time
	FloorID          string
	ZoneID           string
	RequiredFeatures []domain.DeskFeature
}

// ListReservationsInput contains validated reservation list filters.
type ListReservationsInput struct {
	UserID   string
	DeskID   string
	FloorID  string
	Status   domain.ReservationStatus
	StartsAt *time.Time
	EndsAt   *time.Time
}

// CancelReservationInput contains validated data for cancellation.
type CancelReservationInput struct {
	ReservationID string
	ActorUserID   string
	ActorRole     string
}

// ReleaseReservationInput contains validated data for early release.
type ReleaseReservationInput struct {
	ReservationID string
	Reason        string
	ActorUserID   string
	ActorRole     string
}

// NewReservationService creates a reservation service.
func NewReservationService(
	reservations repository.ReservationRepository,
	desks repository.DeskRepository,
	users repository.UserRepository,
) *ReservationService {
	return &ReservationService{
		reservations: reservations,
		desks:        desks,
		users:        users,
		now:          time.Now,
	}
}

// Create creates a manual reservation after verifying related entities.
func (service *ReservationService) Create(ctx context.Context, input CreateReservationInput) (ReservationDetails, error) {
	desk, err := service.desks.GetByID(ctx, input.DeskID)
	if err != nil {
		return ReservationDetails{}, err
	}

	if desk.State != domain.DeskStateActive {
		return ReservationDetails{}, fmt.Errorf("desk is not available for booking: %w", domain.ErrConflict)
	}

	user, err := service.users.GetByID(ctx, input.UserID)
	if err != nil {
		return ReservationDetails{}, err
	}

	reservation := domain.Reservation{
		DeskID:     input.DeskID,
		UserID:     input.UserID,
		Source:     domain.ReservationSourceManual,
		Status:     domain.ReservationStatusActive,
		HolderName: strings.TrimSpace(user.FullName),
		Note:       strings.TrimSpace(input.Note),
		StartsAt:   input.StartsAt.UTC(),
		EndsAt:     input.EndsAt.UTC(),
	}

	createdReservation, err := service.reservations.Create(ctx, reservation)
	if err != nil {
		return ReservationDetails{}, err
	}

	return toReservationDetails(createdReservation, desk), nil
}

// CreateAuto selects the first suitable active desk and creates an automatic reservation.
func (service *ReservationService) CreateAuto(ctx context.Context, input AutoReservationInput) (ReservationDetails, error) {
	selectedDesk, err := service.selectAutoDesk(ctx, input)
	if err != nil {
		return ReservationDetails{}, err
	}

	user, err := service.users.GetByID(ctx, input.UserID)
	if err != nil {
		return ReservationDetails{}, err
	}

	reservation := domain.Reservation{
		DeskID:     selectedDesk.ID,
		UserID:     input.UserID,
		Source:     domain.ReservationSourceAuto,
		Status:     domain.ReservationStatusActive,
		HolderName: strings.TrimSpace(user.FullName),
		StartsAt:   input.StartsAt.UTC(),
		EndsAt:     input.EndsAt.UTC(),
	}

	createdReservation, err := service.reservations.Create(ctx, reservation)
	if err != nil {
		return ReservationDetails{}, err
	}

	return toReservationDetails(createdReservation, selectedDesk), nil
}

// PreviewAuto selects the first suitable active desk without creating a reservation.
func (service *ReservationService) PreviewAuto(ctx context.Context, input AutoReservationInput) (AutoReservationPreview, error) {
	selectedDesk, err := service.selectAutoDesk(ctx, input)
	if err != nil {
		return AutoReservationPreview{}, err
	}

	reasons := make([]string, 0, 3)
	if input.FloorID != "" {
		reasons = append(reasons, fmt.Sprintf("matched floorId=%s", input.FloorID))
	}
	if input.ZoneID != "" {
		reasons = append(reasons, fmt.Sprintf("matched zoneId=%s", input.ZoneID))
	}
	if len(input.RequiredFeatures) > 0 {
		reasons = append(reasons, fmt.Sprintf("matched %d required feature(s)", len(input.RequiredFeatures)))
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "selected first active desk by label")
	}

	return AutoReservationPreview{
		Desk:    selectedDesk,
		Score:   1,
		Reasons: reasons,
	}, nil
}

// List returns reservations matching the provided filters.
func (service *ReservationService) List(ctx context.Context, input ListReservationsInput) ([]ReservationDetails, error) {
	reservations, err := service.reservations.List(ctx, repository.ReservationFilter{
		UserID:   input.UserID,
		DeskID:   input.DeskID,
		FloorID:  input.FloorID,
		Status:   input.Status,
		StartsAt: input.StartsAt,
		EndsAt:   input.EndsAt,
	})
	if err != nil {
		return nil, err
	}

	if len(reservations) == 0 {
		return nil, nil
	}

	uniqueDeskIDs := make([]string, 0, len(reservations))
	seen := make(map[string]struct{}, len(reservations))
	for _, r := range reservations {
		if _, exists := seen[r.DeskID]; !exists {
			seen[r.DeskID] = struct{}{}
			uniqueDeskIDs = append(uniqueDeskIDs, r.DeskID)
		}
	}

	desks, err := service.desks.ListByIDs(ctx, uniqueDeskIDs)
	if err != nil {
		return nil, err
	}

	deskByID := make(map[string]domain.Desk, len(desks))
	for _, desk := range desks {
		deskByID[desk.ID] = desk
	}

	result := make([]ReservationDetails, 0, len(reservations))
	for _, reservation := range reservations {
		result = append(result, toReservationDetails(reservation, deskByID[reservation.DeskID]))
	}

	return result, nil
}

// Get returns a reservation by identifier.
func (service *ReservationService) Get(ctx context.Context, reservationID string) (ReservationDetails, error) {
	reservation, err := service.reservations.GetByID(ctx, reservationID)
	if err != nil {
		return ReservationDetails{}, err
	}

	desk, err := service.desks.GetByID(ctx, reservation.DeskID)
	if err != nil {
		return ReservationDetails{}, err
	}

	return toReservationDetails(reservation, desk), nil
}

// Cancel cancels an active reservation and returns the updated entity snapshot.
func (service *ReservationService) Cancel(ctx context.Context, input CancelReservationInput) (ReservationDetails, error) {
	reservation, err := service.reservations.GetByID(ctx, input.ReservationID)
	if err != nil {
		return ReservationDetails{}, err
	}

	if input.ActorRole != string(domain.UserRoleAdmin) && reservation.UserID != input.ActorUserID {
		return ReservationDetails{}, fmt.Errorf("reservation belongs to another user: %w", domain.ErrForbidden)
	}

	cancelledAt := service.now().UTC()
	if err := service.reservations.Cancel(ctx, input.ReservationID, cancelledAt); err != nil {
		return ReservationDetails{}, err
	}

	reservation, err = service.reservations.GetByID(ctx, input.ReservationID)
	if err != nil {
		return ReservationDetails{}, err
	}

	desk, err := service.desks.GetByID(ctx, reservation.DeskID)
	if err != nil {
		return ReservationDetails{}, err
	}

	return toReservationDetails(reservation, desk), nil
}

// Release completes an active reservation early and returns the updated entity snapshot.
func (service *ReservationService) Release(ctx context.Context, input ReleaseReservationInput) (ReservationDetails, error) {
	reservation, err := service.reservations.GetByID(ctx, input.ReservationID)
	if err != nil {
		return ReservationDetails{}, err
	}

	if input.ActorRole != string(domain.UserRoleAdmin) && reservation.UserID != input.ActorUserID {
		return ReservationDetails{}, fmt.Errorf("reservation belongs to another user: %w", domain.ErrForbidden)
	}

	releasedAt := service.now().UTC()
	if err := service.reservations.Release(ctx, input.ReservationID, releasedAt, strings.TrimSpace(input.Reason)); err != nil {
		return ReservationDetails{}, err
	}

	reservation, err = service.reservations.GetByID(ctx, input.ReservationID)
	if err != nil {
		return ReservationDetails{}, err
	}

	desk, err := service.desks.GetByID(ctx, reservation.DeskID)
	if err != nil {
		return ReservationDetails{}, err
	}

	return toReservationDetails(reservation, desk), nil
}

// ParseReservationStatus validates and normalizes a reservation status filter.
func ParseReservationStatus(value string) (domain.ReservationStatus, error) {
	normalizedValue := strings.TrimSpace(strings.ToLower(value))
	if normalizedValue == "" || normalizedValue == "all" {
		return "", nil
	}

	switch domain.ReservationStatus(normalizedValue) {
	case domain.ReservationStatusActive, domain.ReservationStatusCancelled, domain.ReservationStatusCompleted:
		return domain.ReservationStatus(normalizedValue), nil
	default:
		return "", fmt.Errorf("status must be one of active, cancelled, completed, all: %w", domain.ErrValidation)
	}
}

func (service *ReservationService) selectAutoDesk(ctx context.Context, input AutoReservationInput) (domain.Desk, error) {
	candidates, err := service.desks.List(ctx, repository.DeskFilter{
		FloorID:    strings.TrimSpace(input.FloorID),
		ZoneID:     strings.TrimSpace(input.ZoneID),
		Features:   input.RequiredFeatures,
		ActiveOnly: true,
	})
	if err != nil {
		return domain.Desk{}, err
	}

	if len(candidates) == 0 {
		return domain.Desk{}, fmt.Errorf("no suitable active desk available for the requested time slot: %w", domain.ErrNotFound)
	}

	// Batch-fetch all active reservations that overlap the requested window for all
	// candidate desks in a single query, then determine availability in memory.
	deskIDs := make([]string, 0, len(candidates))
	for _, desk := range candidates {
		deskIDs = append(deskIDs, desk.ID)
	}

	overlapping, err := service.reservations.List(ctx, repository.ReservationFilter{
		DeskIDs:  deskIDs,
		Status:   domain.ReservationStatusActive,
		StartsAt: &input.StartsAt,
		EndsAt:   &input.EndsAt,
	})
	if err != nil {
		return domain.Desk{}, err
	}

	busyDesks := make(map[string]struct{}, len(overlapping))
	for _, r := range overlapping {
		busyDesks[r.DeskID] = struct{}{}
	}

	for _, desk := range candidates {
		if _, busy := busyDesks[desk.ID]; !busy {
			return desk, nil
		}
	}

	return domain.Desk{}, fmt.Errorf("no suitable active desk available for the requested time slot: %w", domain.ErrNotFound)
}

func toReservationDetails(reservation domain.Reservation, desk domain.Desk) ReservationDetails {
	return ReservationDetails{
		Reservation: reservation,
		DeskLabel:   desk.Label,
		FloorID:     desk.FloorID,
	}
}
