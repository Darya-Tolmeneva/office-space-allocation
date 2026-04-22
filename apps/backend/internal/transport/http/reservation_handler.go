package http

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/pkg/logctx"
	"office-space-allocation/apps/backend/internal/service"
)

// ReservationHandler serves reservation endpoints.
type ReservationHandler struct {
	reservationService reservationService
}

// NewReservationHandler creates a reservation HTTP handler.
func NewReservationHandler(svc reservationService) *ReservationHandler {
	return &ReservationHandler{reservationService: svc}
}

type createReservationRequest struct {
	DeskID string `json:"deskId"`
	From   string `json:"from"`
	To     string `json:"to"`
	Note   string `json:"note"`
}

type autoReservationRequest struct {
	From             string   `json:"from"`
	To               string   `json:"to"`
	FloorID          string   `json:"floorId"`
	ZoneID           string   `json:"zoneId"`
	RequiredFeatures []string `json:"requiredFeatures"`
}

type releaseReservationRequest struct {
	Reason string `json:"reason"`
}

type reservationPayload struct {
	ID          string `json:"id"`
	DeskID      string `json:"deskId"`
	DeskLabel   string `json:"deskLabel"`
	FloorID     string `json:"floorId"`
	From        string `json:"from"`
	To          string `json:"to"`
	Status      string `json:"status"`
	Source      string `json:"source"`
	HolderName  string `json:"holderName,omitempty"`
	Note        string `json:"note,omitempty"`
	CreatedAt   string `json:"createdAt"`
	CancelledAt string `json:"cancelledAt,omitempty"`
}

type autoReservationPreviewPayload struct {
	Desk    deskPayload `json:"desk"`
	Score   float64     `json:"score"`
	Reasons []string    `json:"reasons"`
}

func (handler *ReservationHandler) List(writer http.ResponseWriter, request *http.Request) {
	authenticatedUser, ok := AuthenticatedUserFromContext(request.Context())
	if !ok {
		writeDomainError(writer, request.Context(), fmt.Errorf("authenticated user is required: %w", ErrAuthRequired))
		return
	}

	input, err := validateListReservationsQuery(request)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	// Admins may list all reservations; regular members see only their own.
	if authenticatedUser.Role != "admin" {
		input.UserID = authenticatedUser.UserID
	}

	reservations, err := handler.reservationService.List(request.Context(), input)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	response := make([]reservationPayload, 0, len(reservations))
	for _, reservation := range reservations {
		response = append(response, toReservationPayload(reservation))
	}

	writeJSON(writer, http.StatusOK, response)
}

func (handler *ReservationHandler) Create(writer http.ResponseWriter, request *http.Request) {
	payload, ok := decodeJSON[createReservationRequest](writer, request)
	if !ok {
		return
	}

	authenticatedUser, ok := AuthenticatedUserFromContext(request.Context())
	if !ok {
		writeDomainError(writer, request.Context(), fmt.Errorf("authenticated user is required: %w", ErrAuthRequired))
		return
	}

	input, err := validateCreateReservationRequest(payload, authenticatedUser.UserID)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	logctx.Logger(request.Context()).Info("create reservation", "userId", input.UserID, "deskId", input.DeskID)

	reservation, err := handler.reservationService.Create(request.Context(), input)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusCreated, toReservationPayload(reservation))
}

func (handler *ReservationHandler) CreateAuto(writer http.ResponseWriter, request *http.Request) {
	payload, ok := decodeJSON[autoReservationRequest](writer, request)
	if !ok {
		return
	}

	authenticatedUser, ok := AuthenticatedUserFromContext(request.Context())
	if !ok {
		writeDomainError(writer, request.Context(), fmt.Errorf("authenticated user is required: %w", ErrAuthRequired))
		return
	}

	input, err := validateAutoReservationRequest(payload, authenticatedUser.UserID)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	logctx.Logger(request.Context()).Info("create auto reservation", "userId", input.UserID)

	reservation, err := handler.reservationService.CreateAuto(request.Context(), input)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusCreated, toReservationPayload(reservation))
}

func (handler *ReservationHandler) PreviewAuto(writer http.ResponseWriter, request *http.Request) {
	payload, ok := decodeJSON[autoReservationRequest](writer, request)
	if !ok {
		return
	}

	authenticatedUser, ok := AuthenticatedUserFromContext(request.Context())
	if !ok {
		writeDomainError(writer, request.Context(), fmt.Errorf("authenticated user is required: %w", ErrAuthRequired))
		return
	}

	input, err := validateAutoReservationRequest(payload, authenticatedUser.UserID)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	preview, err := handler.reservationService.PreviewAuto(request.Context(), input)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusOK, autoReservationPreviewPayload{
		Desk:    toDeskPayload(preview.Desk),
		Score:   preview.Score,
		Reasons: preview.Reasons,
	})
}

func (handler *ReservationHandler) Get(writer http.ResponseWriter, request *http.Request) {
	reservationID, err := validateReservationIDParam(request)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	reservation, err := handler.reservationService.Get(request.Context(), reservationID)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusOK, toReservationPayload(reservation))
}

func (handler *ReservationHandler) Cancel(writer http.ResponseWriter, request *http.Request) {
	authenticatedUser, ok := AuthenticatedUserFromContext(request.Context())
	if !ok {
		writeDomainError(writer, request.Context(), fmt.Errorf("authenticated user is required: %w", ErrAuthRequired))
		return
	}

	reservationID, err := validateReservationIDParam(request)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	logctx.Logger(request.Context()).Info("cancel reservation", "userId", authenticatedUser.UserID, "reservationId", reservationID)

	if _, err := handler.reservationService.Cancel(request.Context(), service.CancelReservationInput{
		ReservationID: reservationID,
		ActorUserID:   authenticatedUser.UserID,
		ActorRole:     authenticatedUser.Role,
	}); err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (handler *ReservationHandler) Release(writer http.ResponseWriter, request *http.Request) {
	authenticatedUser, ok := AuthenticatedUserFromContext(request.Context())
	if !ok {
		writeDomainError(writer, request.Context(), fmt.Errorf("authenticated user is required: %w", ErrAuthRequired))
		return
	}

	reservationID, err := validateReservationIDParam(request)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	payload, ok := decodeOptionalJSON[releaseReservationRequest](writer, request)
	if !ok {
		return
	}

	input, err := validateReleaseReservationRequest(reservationID, payload, authenticatedUser)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	logctx.Logger(request.Context()).Info("release reservation", "userId", authenticatedUser.UserID, "reservationId", reservationID)

	reservation, err := handler.reservationService.Release(request.Context(), input)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusOK, toReservationPayload(reservation))
}

func validateCreateReservationRequest(payload createReservationRequest, userID string) (service.CreateReservationInput, error) {
	deskID := strings.TrimSpace(payload.DeskID)
	actorUserID := strings.TrimSpace(userID)
	note := strings.TrimSpace(payload.Note)

	if actorUserID == "" {
		return service.CreateReservationInput{}, fmt.Errorf("authenticated user id is required: %w", ErrAuthRequired)
	}

	if deskID == "" {
		return service.CreateReservationInput{}, fmt.Errorf("deskId is required: %w", domain.ErrValidation)
	}

	startsAt, err := parseRequiredTimestamp("from", payload.From)
	if err != nil {
		return service.CreateReservationInput{}, err
	}

	endsAt, err := parseRequiredTimestamp("to", payload.To)
	if err != nil {
		return service.CreateReservationInput{}, err
	}

	if !startsAt.Before(endsAt) {
		return service.CreateReservationInput{}, fmt.Errorf("from must be earlier than to: %w", domain.ErrValidation)
	}

	return service.CreateReservationInput{
		DeskID:   deskID,
		UserID:   actorUserID,
		Note:     note,
		StartsAt: startsAt,
		EndsAt:   endsAt,
	}, nil
}

func validateAutoReservationRequest(payload autoReservationRequest, userID string) (service.AutoReservationInput, error) {
	actorUserID := strings.TrimSpace(userID)
	if actorUserID == "" {
		return service.AutoReservationInput{}, fmt.Errorf("authenticated user id is required: %w", ErrAuthRequired)
	}

	startsAt, err := parseRequiredTimestamp("from", payload.From)
	if err != nil {
		return service.AutoReservationInput{}, err
	}

	endsAt, err := parseRequiredTimestamp("to", payload.To)
	if err != nil {
		return service.AutoReservationInput{}, err
	}

	if !startsAt.Before(endsAt) {
		return service.AutoReservationInput{}, fmt.Errorf("from must be earlier than to: %w", domain.ErrValidation)
	}

	requiredFeatures, err := parseAutoReservationFeatures(payload.RequiredFeatures)
	if err != nil {
		return service.AutoReservationInput{}, err
	}

	return service.AutoReservationInput{
		UserID:           actorUserID,
		StartsAt:         startsAt,
		EndsAt:           endsAt,
		FloorID:          strings.TrimSpace(payload.FloorID),
		ZoneID:           strings.TrimSpace(payload.ZoneID),
		RequiredFeatures: requiredFeatures,
	}, nil
}

func validateListReservationsQuery(request *http.Request) (service.ListReservationsInput, error) {
	query := request.URL.Query()
	status, err := service.ParseReservationStatus(query.Get("status"))
	if err != nil {
		return service.ListReservationsInput{}, err
	}

	startsAt, err := parseOptionalTimestamp("from", query.Get("from"))
	if err != nil {
		return service.ListReservationsInput{}, err
	}

	endsAt, err := parseOptionalTimestamp("to", query.Get("to"))
	if err != nil {
		return service.ListReservationsInput{}, err
	}

	if startsAt != nil && endsAt != nil && !startsAt.Before(*endsAt) {
		return service.ListReservationsInput{}, fmt.Errorf("from must be earlier than to: %w", domain.ErrValidation)
	}

	return service.ListReservationsInput{
		DeskID:   strings.TrimSpace(query.Get("deskId")),
		FloorID:  strings.TrimSpace(query.Get("floorId")),
		Status:   status,
		StartsAt: startsAt,
		EndsAt:   endsAt,
	}, nil
}

func validateReleaseReservationRequest(reservationID string, payload releaseReservationRequest, actor AuthenticatedUser) (service.ReleaseReservationInput, error) {
	return service.ReleaseReservationInput{
		ReservationID: reservationID,
		Reason:        strings.TrimSpace(payload.Reason),
		ActorUserID:   actor.UserID,
		ActorRole:     actor.Role,
	}, nil
}

func validateReservationIDParam(request *http.Request) (string, error) {
	reservationID := strings.TrimSpace(chi.URLParam(request, "reservationId"))
	if reservationID == "" {
		return "", fmt.Errorf("reservationId is required: %w", domain.ErrValidation)
	}

	return reservationID, nil
}

func parseRequiredTimestamp(fieldName string, value string) (time.Time, error) {
	timestamp, err := parseTimestamp(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be a valid RFC3339 date-time: %w", fieldName, domain.ErrValidation)
	}

	return timestamp, nil
}

func parseOptionalTimestamp(fieldName string, value string) (*time.Time, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil, nil
	}

	timestamp, err := parseTimestamp(trimmedValue)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid RFC3339 date-time: %w", fieldName, domain.ErrValidation)
	}

	return &timestamp, nil
}

func parseTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.TrimSpace(value))
}

func parseAutoReservationFeatures(values []string) ([]domain.DeskFeature, error) {
	features := make([]domain.DeskFeature, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}

		feature, err := parseDeskFeature(value)
		if err != nil {
			return nil, fmt.Errorf("requiredFeatures contains %w", err)
		}

		features = append(features, feature)
	}

	return features, nil
}

func decodeOptionalJSON[T any](writer http.ResponseWriter, request *http.Request) (T, bool) {
	var payload T
	if request.Body == nil {
		return payload, true
	}

	if request.ContentLength == 0 {
		return payload, true
	}

	return decodeJSON[T](writer, request)
}

func toReservationPayload(details service.ReservationDetails) reservationPayload {
	reservation := details.Reservation
	payload := reservationPayload{
		ID:         reservation.ID,
		DeskID:     reservation.DeskID,
		DeskLabel:  details.DeskLabel,
		FloorID:    details.FloorID,
		From:       reservation.StartsAt.UTC().Format(time.RFC3339),
		To:         reservation.EndsAt.UTC().Format(time.RFC3339),
		Status:     string(reservation.Status),
		Source:     string(reservation.Source),
		HolderName: reservation.HolderName,
		Note:       reservation.Note,
		CreatedAt:  reservation.CreatedAt.UTC().Format(time.RFC3339),
	}

	if reservation.CancelledAt != nil {
		payload.CancelledAt = reservation.CancelledAt.UTC().Format(time.RFC3339)
	}

	return payload
}
