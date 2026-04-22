package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/pkg/logctx"
)

var validDeskFeatures = map[string]domain.DeskFeature{
	"monitor":      domain.DeskFeatureMonitor,
	"dual_monitor": domain.DeskFeatureDualMonitor,
	"wifi":         domain.DeskFeatureWiFi,
	"ethernet":     domain.DeskFeatureEthernet,
	"standing":     domain.DeskFeatureStanding,
	"quiet":        domain.DeskFeatureQuiet,
	"window":       domain.DeskFeatureWindow,
	"near_kitchen": domain.DeskFeatureNearKitchen,
	"accessible":   domain.DeskFeatureAccessible,
}

// parseDeskFeature validates a single feature string and returns the domain value.
func parseDeskFeature(value string) (domain.DeskFeature, error) {
	key := strings.ToLower(strings.TrimSpace(value))
	if feature, ok := validDeskFeatures[key]; ok {
		return feature, nil
	}

	return "", fmt.Errorf("unsupported desk feature %q: %w", value, domain.ErrValidation)
}

type errorResponse struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeDomainError(writer http.ResponseWriter, ctx context.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		writeJSON(writer, http.StatusBadRequest, errorResponse{
			Error: errorPayload{Code: "validation_error", Message: unwrapMessage(err)},
		})
	case errors.Is(err, ErrAuthRequired):
		writeJSON(writer, http.StatusUnauthorized, errorResponse{
			Error: errorPayload{Code: "unauthorized", Message: unwrapMessage(err)},
		})
	case errors.Is(err, domain.ErrUnauthorized):
		writeJSON(writer, http.StatusUnauthorized, errorResponse{
			Error: errorPayload{Code: "unauthorized", Message: "invalid email or password"},
		})
	case errors.Is(err, domain.ErrForbidden):
		writeJSON(writer, http.StatusForbidden, errorResponse{
			Error: errorPayload{Code: "forbidden", Message: unwrapMessage(err)},
		})
	case errors.Is(err, domain.ErrConflict):
		writeJSON(writer, http.StatusConflict, errorResponse{
			Error: errorPayload{Code: "conflict", Message: unwrapMessage(err)},
		})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(writer, http.StatusNotFound, errorResponse{
			Error: errorPayload{Code: "not_found", Message: unwrapMessage(err)},
		})
	default:
		logctx.Logger(ctx).Error("internal server error", "error", err)
		writeJSON(writer, http.StatusInternalServerError, errorResponse{
			Error: errorPayload{Code: "internal_error", Message: "internal server error"},
		})
	}
}

// unwrapMessage returns the human-readable part of the error message by
// stripping any appended sentinel suffix (": not found", ": conflict", etc.).
func unwrapMessage(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	for _, sentinel := range []error{
		domain.ErrValidation,
		domain.ErrNotFound,
		domain.ErrConflict,
		domain.ErrForbidden,
		domain.ErrUnauthorized,
		ErrAuthRequired,
	} {
		suffix := ": " + sentinel.Error()
		if strings.HasSuffix(msg, suffix) {
			return strings.TrimSuffix(msg, suffix)
		}
	}

	return msg
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		// Programming bug — payload contains a non-serialisable type.
		logctx.Logger(context.Background()).Error("writeJSON marshal error", "error", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_, _ = writer.Write(data)
}

func decodeJSON[T any](writer http.ResponseWriter, request *http.Request) (T, bool) {
	var payload T

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError

		var msg string
		switch {
		case errors.As(err, &syntaxErr):
			msg = fmt.Sprintf("request body contains malformed JSON at position %d", syntaxErr.Offset)
		case errors.As(err, &typeErr):
			msg = fmt.Sprintf("field %q must be of type %s", typeErr.Field, typeErr.Type)
		default:
			msg = "request body must be valid JSON"
		}

		writeJSON(writer, http.StatusBadRequest, errorResponse{
			Error: errorPayload{Code: "bad_request", Message: msg},
		})
		return payload, false
	}

	if decoder.More() {
		writeJSON(writer, http.StatusBadRequest, errorResponse{
			Error: errorPayload{Code: "bad_request", Message: "request body must contain a single JSON object"},
		})
		return payload, false
	}

	return payload, true
}
