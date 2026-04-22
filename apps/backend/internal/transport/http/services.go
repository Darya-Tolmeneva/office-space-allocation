package http

import (
	"context"
	"time"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/service"
)

// authService defines authentication operations consumed by the transport layer.
type authService interface {
	Register(ctx context.Context, input service.RegisterInput) (service.AuthResult, error)
	Login(ctx context.Context, input service.LoginInput) (service.AuthResult, error)
	Refresh(ctx context.Context, rawRefreshToken string) (service.AuthResult, error)
}

// workspaceService defines catalog read operations consumed by the transport layer.
type workspaceService interface {
	ListFloors(ctx context.Context) ([]domain.Floor, error)
	GetFloorDetails(ctx context.Context, floorID string) (service.FloorDetails, error)
	ListDesks(ctx context.Context, input service.ListDesksInput) ([]domain.Desk, error)
	GetDesk(ctx context.Context, deskID string) (domain.Desk, error)
	GetDeskAvailability(ctx context.Context, deskID string, startsAt time.Time, endsAt time.Time) ([]service.DeskAvailabilitySlot, error)
}

// reservationService defines reservation operations consumed by the transport layer.
type reservationService interface {
	Create(ctx context.Context, input service.CreateReservationInput) (service.ReservationDetails, error)
	CreateAuto(ctx context.Context, input service.AutoReservationInput) (service.ReservationDetails, error)
	PreviewAuto(ctx context.Context, input service.AutoReservationInput) (service.AutoReservationPreview, error)
	List(ctx context.Context, input service.ListReservationsInput) ([]service.ReservationDetails, error)
	Get(ctx context.Context, reservationID string) (service.ReservationDetails, error)
	Cancel(ctx context.Context, input service.CancelReservationInput) (service.ReservationDetails, error)
	Release(ctx context.Context, input service.ReleaseReservationInput) (service.ReservationDetails, error)
}

// analyticsService defines analytics operations consumed by the transport layer.
type analyticsService interface {
	GetSummary(ctx context.Context, input service.AnalyticsSummaryInput) (service.AnalyticsSummary, error)
}
