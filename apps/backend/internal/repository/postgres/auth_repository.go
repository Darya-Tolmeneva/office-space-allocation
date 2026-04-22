package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/pkg/logctx"
)

// UserRepository provides PostgreSQL access to users.
type UserRepository struct {
	database *sqlx.DB
}

// NewUserRepository creates a user repository backed by PostgreSQL.
func NewUserRepository(database *sqlx.DB) *UserRepository {
	return &UserRepository{database: database}
}

// RefreshTokenRepository provides PostgreSQL access to refresh tokens.
type RefreshTokenRepository struct {
	database *sqlx.DB
}

// NewRefreshTokenRepository creates a refresh token repository backed by PostgreSQL.
func NewRefreshTokenRepository(database *sqlx.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{database: database}
}

type userRow struct {
	ID           string    `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	FullName     string    `db:"full_name"`
	Role         string    `db:"role"`
	Status       string    `db:"status"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type refreshTokenRow struct {
	ID        string       `db:"id"`
	UserID    string       `db:"user_id"`
	TokenHash string       `db:"token_hash"`
	ExpiresAt time.Time    `db:"expires_at"`
	RevokedAt sql.NullTime `db:"revoked_at"`
	CreatedAt time.Time    `db:"created_at"`
}

// Create stores a user and returns the persisted entity.
func (repository *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	logctx.Logger(ctx).Info("create user", "email", user.Email)

	const insertQuery = `
		INSERT INTO users (
			email,
			password_hash,
			full_name,
			role,
			status
		)
		VALUES (
			:email,
			:password_hash,
			:full_name,
			:role,
			:status
		)
		RETURNING
			id,
			email,
			password_hash,
			full_name,
			role,
			status,
			created_at,
			updated_at
	`

	params := map[string]any{
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"full_name":     user.FullName,
		"role":          user.Role,
		"status":        user.Status,
	}

	boundQuery, arguments, err := sqlx.Named(insertQuery, params)
	if err != nil {
		return domain.User{}, fmt.Errorf("build create user query: %w", err)
	}

	reboundQuery := repository.database.Rebind(boundQuery)

	var row userRow
	if err := repository.database.GetContext(ctx, &row, reboundQuery, arguments...); err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, domain.ErrConflict
		}

		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return row.toDomain(), nil
}

// GetByEmail returns a user by email.
func (repository *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	const query = `
		SELECT
			id,
			email,
			password_hash,
			full_name,
			role,
			status,
			created_at,
			updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`

	var row userRow
	if err := repository.database.GetContext(ctx, &row, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}

		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return row.toDomain(), nil
}

// GetByID returns a user by identifier.
func (repository *UserRepository) GetByID(ctx context.Context, userID string) (domain.User, error) {
	const query = `
		SELECT
			id,
			email,
			password_hash,
			full_name,
			role,
			status,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	var row userRow
	if err := repository.database.GetContext(ctx, &row, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}

		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return row.toDomain(), nil
}

// Create stores a refresh token and returns the persisted entity.
func (repository *RefreshTokenRepository) Create(ctx context.Context, token domain.RefreshToken) (domain.RefreshToken, error) {
	const insertQuery = `
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at,
			revoked_at
		)
		VALUES (
			:user_id,
			:token_hash,
			:expires_at,
			:revoked_at
		)
		RETURNING
			id,
			user_id,
			token_hash,
			expires_at,
			revoked_at,
			created_at
	`

	params := map[string]any{
		"user_id":    token.UserID,
		"token_hash": token.TokenHash,
		"expires_at": token.ExpiresAt,
		"revoked_at": token.RevokedAt,
	}

	boundQuery, arguments, err := sqlx.Named(insertQuery, params)
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("build create refresh token query: %w", err)
	}

	reboundQuery := repository.database.Rebind(boundQuery)

	var row refreshTokenRow
	if err := repository.database.GetContext(ctx, &row, reboundQuery, arguments...); err != nil {
		if isUniqueViolation(err) {
			return domain.RefreshToken{}, domain.ErrConflict
		}

		return domain.RefreshToken{}, fmt.Errorf("create refresh token: %w", err)
	}

	return row.toDomain(), nil
}

// GetByTokenHash returns a refresh token by hash.
func (repository *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	const query = `
		SELECT
			id,
			user_id,
			token_hash,
			expires_at,
			revoked_at,
			created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var row refreshTokenRow
	if err := repository.database.GetContext(ctx, &row, query, tokenHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.RefreshToken{}, domain.ErrNotFound
		}

		return domain.RefreshToken{}, fmt.Errorf("get refresh token by hash: %w", err)
	}

	return row.toDomain(), nil
}

// Revoke marks a refresh token as revoked.
func (repository *RefreshTokenRepository) Revoke(ctx context.Context, tokenID string, revokedAt time.Time) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`

	result, err := repository.database.ExecContext(ctx, query, tokenID, revokedAt)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read revoke refresh token result: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (row userRow) toDomain() domain.User {
	return domain.User{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		FullName:     row.FullName,
		Role:         domain.UserRole(row.Role),
		Status:       domain.UserStatus(row.Status),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func (row refreshTokenRow) toDomain() domain.RefreshToken {
	token := domain.RefreshToken{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}

	if row.RevokedAt.Valid {
		revokedAt := row.RevokedAt.Time
		token.RevokedAt = &revokedAt
	}

	return token
}

func isUniqueViolation(err error) bool {
	var postgresError *pq.Error
	if !errors.As(err, &postgresError) {
		return false
	}

	return postgresError.Code == "23505"
}
