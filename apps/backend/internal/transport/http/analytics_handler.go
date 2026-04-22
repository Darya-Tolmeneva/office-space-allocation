package http

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/service"
)

// AnalyticsHandler serves analytics endpoints.
type AnalyticsHandler struct {
	analyticsService analyticsService
}

// NewAnalyticsHandler creates an analytics HTTP handler.
func NewAnalyticsHandler(svc analyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: svc}
}

type analyticsSummaryPayload struct {
	AverageOccupancy  float64                 `json:"averageOccupancy"`
	PeakDay           string                  `json:"peakDay,omitempty"`
	PeakOccupancy     float64                 `json:"peakOccupancy"`
	AutoPickRatio     float64                 `json:"autoPickRatio"`
	TotalReservations int                     `json:"totalReservations"`
	EarlyReleases     int                     `json:"earlyReleases"`
	TopZone           analyticsTopZonePayload `json:"topZone"`
}

type analyticsTopZonePayload struct {
	ZoneID string `json:"zoneId,omitempty"`
	Name   string `json:"name,omitempty"`
}

func (handler *AnalyticsHandler) Summary(writer http.ResponseWriter, request *http.Request) {
	input, err := validateAnalyticsSummaryQuery(request)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	summary, err := handler.analyticsService.GetSummary(request.Context(), input)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusOK, analyticsSummaryPayload{
		AverageOccupancy:  summary.AverageOccupancy,
		PeakDay:           summary.PeakDay,
		PeakOccupancy:     summary.PeakOccupancy,
		AutoPickRatio:     summary.AutoPickRatio,
		TotalReservations: summary.TotalReservations,
		EarlyReleases:     summary.EarlyReleases,
		TopZone: analyticsTopZonePayload{
			ZoneID: summary.TopZone.ZoneID,
			Name:   summary.TopZone.Name,
		},
	})
}

func validateAnalyticsSummaryQuery(request *http.Request) (service.AnalyticsSummaryInput, error) {
	query := request.URL.Query()
	from, err := parseOptionalDate("from", query.Get("from"))
	if err != nil {
		return service.AnalyticsSummaryInput{}, err
	}

	to, err := parseOptionalDate("to", query.Get("to"))
	if err != nil {
		return service.AnalyticsSummaryInput{}, err
	}

	return service.AnalyticsSummaryInput{
		FloorID: strings.TrimSpace(query.Get("floorId")),
		From:    from,
		To:      to,
	}, nil
}

func parseOptionalDate(fieldName string, value string) (*time.Time, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil, nil
	}

	parsedDate, err := time.Parse("2006-01-02", trimmedValue)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid date in YYYY-MM-DD format: %w", fieldName, domain.ErrValidation)
	}

	parsedDate = parsedDate.UTC()
	return &parsedDate, nil
}
