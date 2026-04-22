package domain

import "time"

// ReservationSource describes how a reservation was created.
type ReservationSource string

const (
	ReservationSourceManual ReservationSource = "manual"
	ReservationSourceAuto   ReservationSource = "auto"
)

// ReservationStatus describes the lifecycle state of a reservation.
type ReservationStatus string

const (
	ReservationStatusActive    ReservationStatus = "active"
	ReservationStatusCancelled ReservationStatus = "cancelled"
	ReservationStatusCompleted ReservationStatus = "completed"
)

// Reservation represents a desk booking owned by a user.
type Reservation struct {
	ID            string
	DeskID        string
	UserID        string
	Source        ReservationSource
	Status        ReservationStatus
	HolderName    string
	Note          string
	StartsAt      time.Time
	EndsAt        time.Time
	ReleasedAt    *time.Time
	CancelledAt   *time.Time
	ReleaseReason string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
