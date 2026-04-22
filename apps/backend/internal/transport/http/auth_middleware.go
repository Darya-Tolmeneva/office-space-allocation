package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	pkgauth "office-space-allocation/apps/backend/internal/pkg/auth"
)

type authContextKey string

const authenticatedUserContextKey authContextKey = "authenticated-user"

var ErrAuthRequired = errors.New("auth required")

// AuthenticatedUser contains identity extracted from the access token.
type AuthenticatedUser struct {
	UserID string
	Role   string
}

// RequireAuth validates bearer access tokens and stores user identity in request context.
func RequireAuth(signingKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			tokenString, err := extractBearerToken(request.Header.Get("Authorization"))
			if err != nil {
				writeDomainError(writer, request.Context(), err)
				return
			}

			claims, err := parseAccessToken(tokenString, signingKey)
			if err != nil {
				writeDomainError(writer, request.Context(), err)
				return
			}

			contextWithUser := context.WithValue(request.Context(), authenticatedUserContextKey, AuthenticatedUser{
				UserID: claims.UserID,
				Role:   claims.Role,
			})

			next.ServeHTTP(writer, request.WithContext(contextWithUser))
		})
	}
}

// AuthenticatedUserFromContext returns the authenticated user stored by middleware.
func AuthenticatedUserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(authenticatedUserContextKey).(AuthenticatedUser)
	return user, ok
}

func extractBearerToken(headerValue string) (string, error) {
	trimmedValue := strings.TrimSpace(headerValue)
	if trimmedValue == "" {
		return "", fmt.Errorf("authorization header is required: %w", ErrAuthRequired)
	}

	parts := strings.SplitN(trimmedValue, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("authorization header must use Bearer token: %w", ErrAuthRequired)
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("bearer token is required: %w", ErrAuthRequired)
	}

	return token, nil
}

func parseAccessToken(tokenString string, signingKey string) (pkgauth.Claims, error) {
	claims := pkgauth.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}

		return []byte(signingKey), nil
	})
	if err != nil {
		return pkgauth.Claims{}, fmt.Errorf("invalid access token: %w", ErrAuthRequired)
	}

	if !token.Valid {
		return pkgauth.Claims{}, fmt.Errorf("invalid access token: %w", ErrAuthRequired)
	}

	if strings.TrimSpace(claims.UserID) == "" {
		return pkgauth.Claims{}, fmt.Errorf("access token missing user id: %w", ErrAuthRequired)
	}

	return claims, nil
}
