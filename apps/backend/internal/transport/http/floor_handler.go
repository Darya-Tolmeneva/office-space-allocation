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


// FloorHandler serves floor endpoints.
type FloorHandler struct {
	workspaceService workspaceService
}

// NewFloorHandler creates a floor HTTP handler.
func NewFloorHandler(svc workspaceService) *FloorHandler {
	return &FloorHandler{workspaceService: svc}
}

type floorPayload struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Timezone     string        `json:"timezone"`
	FloorPlanURL string        `json:"floorPlanUrl,omitempty"`
	CreatedAt    string        `json:"createdAt"`
	Zones        []zonePayload `json:"zones,omitempty"`
	Desks        []deskPayload `json:"desks,omitempty"`
}

type zonePayload struct {
	ID        string `json:"id"`
	FloorID   string `json:"floorId"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedAt string `json:"createdAt"`
}

func (handler *FloorHandler) List(writer http.ResponseWriter, request *http.Request) {
	floors, err := handler.workspaceService.ListFloors(request.Context())
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	response := make([]floorPayload, 0, len(floors))
	for _, floor := range floors {
		response = append(response, toFloorPayload(floor))
	}

	writeJSON(writer, http.StatusOK, response)
}

func (handler *FloorHandler) Get(writer http.ResponseWriter, request *http.Request) {
	floorID := strings.TrimSpace(chi.URLParam(request, "floorId"))
	if floorID == "" {
		writeDomainError(writer, request.Context(), fmt.Errorf("floorId is required: %w", domain.ErrValidation))
		return
	}

	details, err := handler.workspaceService.GetFloorDetails(request.Context(), floorID)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusOK, toFloorDetailsPayload(details))
}

func toFloorPayload(floor domain.Floor) floorPayload {
	return floorPayload{
		ID:           floor.ID,
		Name:         floor.Name,
		Timezone:     floor.Timezone,
		FloorPlanURL: floor.FloorPlanURL,
		CreatedAt:    floor.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toFloorDetailsPayload(details service.FloorDetails) floorPayload {
	response := toFloorPayload(details.Floor)
	response.Zones = make([]zonePayload, 0, len(details.Zones))
	response.Desks = make([]deskPayload, 0, len(details.Desks))

	for _, zone := range details.Zones {
		response.Zones = append(response.Zones, zonePayload{
			ID:        zone.ID,
			FloorID:   zone.FloorID,
			Name:      zone.Name,
			Type:      string(zone.Type),
			CreatedAt: zone.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	for _, desk := range details.Desks {
		response.Desks = append(response.Desks, toDeskPayload(desk))
	}

	return response
}
