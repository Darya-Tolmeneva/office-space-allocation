package http

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/service"
)

// DeskHandler serves desk endpoints.
type DeskHandler struct {
	workspaceService workspaceService
}

// NewDeskHandler creates a desk HTTP handler.
func NewDeskHandler(svc workspaceService) *DeskHandler {
	return &DeskHandler{workspaceService: svc}
}

type deskPayload struct {
	ID        string           `json:"id"`
	FloorID   string           `json:"floorId"`
	ZoneID    string           `json:"zoneId,omitempty"`
	Label     string           `json:"label"`
	Status    string           `json:"status"`
	Position  deskPositionJSON `json:"position"`
	Features  []string         `json:"features"`
	CreatedAt string           `json:"createdAt"`
}

type deskPositionJSON struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type availabilitySlotPayload struct {
	From          string `json:"from"`
	To            string `json:"to"`
	Status        string `json:"status"`
	ReservationID string `json:"reservationId,omitempty"`
}

func (handler *DeskHandler) List(writer http.ResponseWriter, request *http.Request) {
	input, err := validateListDesksQuery(request)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	desks, err := handler.workspaceService.ListDesks(request.Context(), input)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	response := make([]deskPayload, 0, len(desks))
	for _, desk := range desks {
		response = append(response, toDeskPayload(desk))
	}

	writeJSON(writer, http.StatusOK, response)
}

func (handler *DeskHandler) Get(writer http.ResponseWriter, request *http.Request) {
	deskID := strings.TrimSpace(chi.URLParam(request, "deskId"))
	if deskID == "" {
		writeDomainError(writer, request.Context(), fmt.Errorf("deskId is required: %w", domain.ErrValidation))
		return
	}

	desk, err := handler.workspaceService.GetDesk(request.Context(), deskID)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusOK, toDeskPayload(desk))
}

func (handler *DeskHandler) Availability(writer http.ResponseWriter, request *http.Request) {
	deskID := strings.TrimSpace(chi.URLParam(request, "deskId"))
	if deskID == "" {
		writeDomainError(writer, request.Context(), fmt.Errorf("deskId is required: %w", domain.ErrValidation))
		return
	}

	startsAt, err := parseRequiredTimestamp("from", request.URL.Query().Get("from"))
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	endsAt, err := parseRequiredTimestamp("to", request.URL.Query().Get("to"))
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	if !startsAt.Before(endsAt) {
		writeDomainError(writer, request.Context(), fmt.Errorf("from must be earlier than to: %w", domain.ErrValidation))
		return
	}

	slots, err := handler.workspaceService.GetDeskAvailability(request.Context(), deskID, startsAt, endsAt)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	response := make([]availabilitySlotPayload, 0, len(slots))
	for _, slot := range slots {
		response = append(response, availabilitySlotPayload{
			From:          slot.From.UTC().Format(time.RFC3339),
			To:            slot.To.UTC().Format(time.RFC3339),
			Status:        slot.Status,
			ReservationID: slot.ReservationID,
		})
	}

	writeJSON(writer, http.StatusOK, response)
}

func validateListDesksQuery(request *http.Request) (service.ListDesksInput, error) {
	query := request.URL.Query()
	features, err := parseDeskFeaturesQuery(query.Get("features"))
	if err != nil {
		return service.ListDesksInput{}, err
	}

	return service.ListDesksInput{
		FloorID:  strings.TrimSpace(query.Get("floorId")),
		ZoneID:   strings.TrimSpace(query.Get("zoneId")),
		Features: features,
	}, nil
}

func parseDeskFeaturesQuery(raw string) ([]domain.DeskFeature, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	features := make([]domain.DeskFeature, 0, len(parts))
	seen := make(map[domain.DeskFeature]struct{}, len(parts))

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}

		feature, err := parseDeskFeature(part)
		if err != nil {
			return nil, err
		}

		if _, exists := seen[feature]; exists {
			continue
		}

		seen[feature] = struct{}{}
		features = append(features, feature)
	}

	return features, nil
}

func toDeskPayload(desk domain.Desk) deskPayload {
	features := make([]string, 0, len(desk.Features))
	for _, feature := range desk.Features {
		features = append(features, string(feature))
	}

	return deskPayload{
		ID:      desk.ID,
		FloorID: desk.FloorID,
		ZoneID:  desk.ZoneID,
		Label:   desk.Label,
		Status:  string(desk.State),
		Position: deskPositionJSON{
			X: desk.Position.X,
			Y: desk.Position.Y,
		},
		Features:  features,
		CreatedAt: desk.CreatedAt.UTC().Format(time.RFC3339),
	}
}
