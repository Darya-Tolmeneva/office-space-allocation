package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	pkgauth "office-space-allocation/apps/backend/internal/pkg/auth"
	"office-space-allocation/apps/backend/internal/pkg/config"
	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/repository"
)

// AuthService handles registration, login, and token refresh flows.
type AuthService struct {
	users         repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	authConfig    config.AuthConfig
	now           func() time.Time
}

// RegisterInput contains validated registration data.
type RegisterInput struct {
	Email    string
	Password string
	FullName string
}

// LoginInput contains validated login data.
type LoginInput struct {
	Email    string
	Password string
}

// AuthResult contains the authenticated user and issued tokens.
type AuthResult struct {
	User   domain.User
	Tokens domain.TokenPair
}

// NewAuthService creates an authentication service.
func NewAuthService(users repository.UserRepository, refreshTokens repository.RefreshTokenRepository, authConfig config.AuthConfig) *AuthService {
	return &AuthService{
		users:         users,
		refreshTokens: refreshTokens,
		authConfig:    authConfig,
		now:           time.Now,
	}
}

// Register creates a new active member account and issues tokens.
func (service *AuthService) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	hashedPassword, err := pkgauth.HashPassword(input.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := service.users.Create(ctx, domain.User{
		Email:        strings.ToLower(input.Email),
		PasswordHash: hashedPassword,
		FullName:     input.FullName,
		Role:         domain.UserRoleMember,
		Status:       domain.UserStatusActive,
	})
	if err != nil {
		return AuthResult{}, err
	}

	return service.issueTokens(ctx, user)
}

// Login authenticates an existing user and issues fresh tokens.
func (service *AuthService) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	user, err := service.users.GetByEmail(ctx, strings.ToLower(input.Email))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return AuthResult{}, domain.ErrUnauthorized
		}

		return AuthResult{}, err
	}

	if user.Status != domain.UserStatusActive {
		return AuthResult{}, domain.ErrForbidden
	}

	if err := pkgauth.ComparePassword(user.PasswordHash, input.Password); err != nil {
		return AuthResult{}, domain.ErrUnauthorized
	}

	return service.issueTokens(ctx, user)
}

func (service *AuthService) issueTokens(ctx context.Context, user domain.User) (AuthResult, error) {
	issuedAt := service.now().UTC()

	accessToken, accessTokenExpiresAt, err := pkgauth.IssueAccessToken(
		service.authConfig.JWTSigningKey,
		user.ID,
		string(user.Role),
		issuedAt,
		service.authConfig.AccessTokenTTL,
	)
	if err != nil {
		return AuthResult{}, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, refreshTokenHash, err := generateRefreshToken()
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshTokenExpiresAt := issuedAt.Add(service.authConfig.RefreshTokenTTL)
	persistedToken, err := service.refreshTokens.Create(ctx, domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: refreshTokenExpiresAt,
	})
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		User: user,
		Tokens: domain.TokenPair{
			AccessToken:           accessToken,
			AccessTokenExpiresAt:  accessTokenExpiresAt,
			RefreshToken:          refreshToken,
			RefreshTokenExpiresAt: persistedToken.ExpiresAt,
		},
	}, nil
}

// Refresh validates an existing refresh token, revokes it, and issues a new token pair.
func (service *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (AuthResult, error) {
	hash := sha256.Sum256([]byte(strings.TrimSpace(rawRefreshToken)))
	tokenHash := hex.EncodeToString(hash[:])

	persistedToken, err := service.refreshTokens.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return AuthResult{}, domain.ErrUnauthorized
		}

		return AuthResult{}, err
	}

	if persistedToken.RevokedAt != nil {
		return AuthResult{}, domain.ErrUnauthorized
	}

	if service.now().After(persistedToken.ExpiresAt) {
		return AuthResult{}, domain.ErrUnauthorized
	}

	if err := service.refreshTokens.Revoke(ctx, persistedToken.ID, service.now().UTC()); err != nil {
		return AuthResult{}, fmt.Errorf("revoke refresh token: %w", err)
	}

	user, err := service.users.GetByID(ctx, persistedToken.UserID)
	if err != nil {
		return AuthResult{}, err
	}

	if user.Status != domain.UserStatusActive {
		return AuthResult{}, domain.ErrForbidden
	}

	return service.issueTokens(ctx, user)
}

func generateRefreshToken() (string, string, error) {
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return "", "", fmt.Errorf("read random bytes: %w", err)
	}

	refreshToken := hex.EncodeToString(rawToken)
	hash := sha256.Sum256([]byte(refreshToken))

	return refreshToken, hex.EncodeToString(hash[:]), nil
}
