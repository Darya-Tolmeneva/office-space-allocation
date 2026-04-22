package http

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/pkg/logctx"
	"office-space-allocation/apps/backend/internal/service"
)


var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// AuthHandler serves authentication endpoints.
type AuthHandler struct {
	authService authService
}

// NewAuthHandler creates an auth HTTP handler.
func NewAuthHandler(svc authService) *AuthHandler {
	return &AuthHandler{authService: svc}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"fullName"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User   userResponse  `json:"user"`
	Tokens tokensPayload `json:"tokens"`
}

type userResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"fullName"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type tokensPayload struct {
	AccessToken           string `json:"accessToken"`
	AccessTokenExpiresAt  string `json:"accessTokenExpiresAt"`
	RefreshToken          string `json:"refreshToken"`
	RefreshTokenExpiresAt string `json:"refreshTokenExpiresAt"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (handler *AuthHandler) Register(writer http.ResponseWriter, request *http.Request) {
	payload, ok := decodeJSON[registerRequest](writer, request)
	if !ok {
		return
	}

	input, err := validateRegisterRequest(payload)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	logctx.Logger(request.Context()).Info("register", "email", input.Email)

	result, err := handler.authService.Register(request.Context(), input)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusCreated, toAuthResponse(result))
}

func (handler *AuthHandler) Refresh(writer http.ResponseWriter, request *http.Request) {
	payload, ok := decodeJSON[refreshRequest](writer, request)
	if !ok {
		return
	}

	if payload.RefreshToken == "" {
		writeDomainError(writer, request.Context(), fmt.Errorf("refreshToken is required: %w", domain.ErrValidation))
		return
	}

	result, err := handler.authService.Refresh(request.Context(), payload.RefreshToken)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusOK, toAuthResponse(result))
}

func (handler *AuthHandler) Login(writer http.ResponseWriter, request *http.Request) {
	payload, ok := decodeJSON[loginRequest](writer, request)
	if !ok {
		return
	}

	input, err := validateLoginRequest(payload)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	logctx.Logger(request.Context()).Info("login", "email", input.Email)

	result, err := handler.authService.Login(request.Context(), input)
	if err != nil {
		writeDomainError(writer, request.Context(), err)
		return
	}

	writeJSON(writer, http.StatusOK, toAuthResponse(result))
}

func validateRegisterRequest(payload registerRequest) (service.RegisterInput, error) {
	email := normalizeEmail(payload.Email)
	fullName := strings.TrimSpace(payload.FullName)
	password := payload.Password

	if email == "" || !emailPattern.MatchString(email) {
		return service.RegisterInput{}, fmt.Errorf("email must be a valid email address: %w", domain.ErrValidation)
	}

	if len(password) < 8 {
		return service.RegisterInput{}, fmt.Errorf("password must be at least 8 characters long: %w", domain.ErrValidation)
	}

	if fullName == "" {
		return service.RegisterInput{}, fmt.Errorf("fullName is required: %w", domain.ErrValidation)
	}

	return service.RegisterInput{
		Email:    email,
		Password: password,
		FullName: fullName,
	}, nil
}

func validateLoginRequest(payload loginRequest) (service.LoginInput, error) {
	email := normalizeEmail(payload.Email)
	password := payload.Password

	if email == "" || !emailPattern.MatchString(email) {
		return service.LoginInput{}, fmt.Errorf("email must be a valid email address: %w", domain.ErrValidation)
	}

	if password == "" {
		return service.LoginInput{}, fmt.Errorf("password is required: %w", domain.ErrValidation)
	}

	return service.LoginInput{
		Email:    email,
		Password: password,
	}, nil
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func toAuthResponse(result service.AuthResult) authResponse {
	return authResponse{
		User: userResponse{
			ID:        result.User.ID,
			Email:     result.User.Email,
			FullName:  result.User.FullName,
			Role:      string(result.User.Role),
			Status:    string(result.User.Status),
			CreatedAt: result.User.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: result.User.UpdatedAt.UTC().Format(time.RFC3339),
		},
		Tokens: tokensPayload{
			AccessToken:           result.Tokens.AccessToken,
			AccessTokenExpiresAt:  result.Tokens.AccessTokenExpiresAt.UTC().Format(time.RFC3339),
			RefreshToken:          result.Tokens.RefreshToken,
			RefreshTokenExpiresAt: result.Tokens.RefreshTokenExpiresAt.UTC().Format(time.RFC3339),
		},
	}
}
