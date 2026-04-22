package repository

import (
	"context"
	"time"

	"office-space-allocation/apps/backend/internal/domain"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user domain.User) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, userID string) (domain.User, error)
}

// RefreshTokenRepository defines persistence operations for refresh tokens.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token domain.RefreshToken) (domain.RefreshToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error)
	Revoke(ctx context.Context, tokenID string, revokedAt time.Time) error
}

// FloorRepository defines read operations for floors.
type FloorRepository interface {
	List(ctx context.Context) ([]domain.Floor, error)
	GetByID(ctx context.Context, floorID string) (domain.Floor, error)
}

// ZoneRepository defines read operations for zones.
type ZoneRepository interface {
	List(ctx context.Context) ([]domain.Zone, error)
	ListByFloorID(ctx context.Context, floorID string) ([]domain.Zone, error)
}

// DeskRepository defines read operations for desks and availability candidates.
type DeskRepository interface {
	List(ctx context.Context, filter DeskFilter) ([]domain.Desk, error)
	GetByID(ctx context.Context, deskID string) (domain.Desk, error)
	ListByIDs(ctx context.Context, deskIDs []string) ([]domain.Desk, error)
	ListAvailability(ctx context.Context, deskID string, startsAt time.Time, endsAt time.Time) ([]AvailabilitySlot, error)
}

// ReservationRepository defines persistence operations for reservations.
type ReservationRepository interface {
	Create(ctx context.Context, reservation domain.Reservation) (domain.Reservation, error)
	GetByID(ctx context.Context, reservationID string) (domain.Reservation, error)
	List(ctx context.Context, filter ReservationFilter) ([]domain.Reservation, error)
	Cancel(ctx context.Context, reservationID string, cancelledAt time.Time) error
	Release(ctx context.Context, reservationID string, releasedAt time.Time, reason string) error
	CompleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// AvailabilitySlot describes desk availability for a requested interval.
type AvailabilitySlot struct {
	From          time.Time
	To            time.Time
	Status        string
	ReservationID string
}

// DeskFilter describes desk query parameters.
type DeskFilter struct {
	FloorID    string
	ZoneID     string
	Features   []domain.DeskFeature
	ActiveOnly bool
}

// ReservationFilter describes reservation query parameters.
type ReservationFilter struct {
	UserID   string
	DeskID   string
	DeskIDs  []string
	FloorID  string
	Status   domain.ReservationStatus
	StartsAt *time.Time
	EndsAt   *time.Time
}
