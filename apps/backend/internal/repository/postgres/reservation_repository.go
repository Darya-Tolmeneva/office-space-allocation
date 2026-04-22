package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"office-space-allocation/apps/backend/internal/domain"
	"office-space-allocation/apps/backend/internal/pkg/logctx"
	"office-space-allocation/apps/backend/internal/repository"
)

// ReservationRepository provides PostgreSQL access to reservations.
type ReservationRepository struct {
	database *sqlx.DB
}

// NewReservationRepository creates a reservation repository backed by PostgreSQL.
func NewReservationRepository(database *sqlx.DB) *ReservationRepository {
	return &ReservationRepository{database: database}
}

type reservationRow struct {
	ID            string         `db:"id"`
	DeskID        string         `db:"desk_id"`
	UserID        string         `db:"user_id"`
	Source        string         `db:"source"`
	Status        string         `db:"status"`
	HolderName    sql.NullString `db:"holder_name"`
	Note          sql.NullString `db:"note"`
	StartsAt      time.Time      `db:"starts_at"`
	EndsAt        time.Time      `db:"ends_at"`
	ReleasedAt    sql.NullTime   `db:"released_at"`
	CancelledAt   sql.NullTime   `db:"cancelled_at"`
	ReleaseReason sql.NullString `db:"release_reason"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
}

// Create stores a reservation and returns the persisted entity.
func (repository *ReservationRepository) Create(ctx context.Context, reservation domain.Reservation) (domain.Reservation, error) {
	logctx.Logger(ctx).Info("insert reservation", "deskId", reservation.DeskID, "userId", reservation.UserID)

	const insertQuery = `
		INSERT INTO reservations (
			desk_id,
			user_id,
			source,
			status,
			holder_name,
			note,
			starts_at,
			ends_at,
			released_at,
			cancelled_at,
			release_reason
		)
		VALUES (
			:desk_id,
			:user_id,
			:source,
			:status,
			:holder_name,
			:note,
			:starts_at,
			:ends_at,
			:released_at,
			:cancelled_at,
			:release_reason
		)
		RETURNING
			id,
			desk_id,
			user_id,
			source,
			status,
			holder_name,
			note,
			starts_at,
			ends_at,
			released_at,
			cancelled_at,
			release_reason,
			created_at,
			updated_at
	`

	params := map[string]any{
		"desk_id":        reservation.DeskID,
		"user_id":        reservation.UserID,
		"source":         reservation.Source,
		"status":         reservation.Status,
		"holder_name":    nullableString(reservation.HolderName),
		"note":           nullableString(reservation.Note),
		"starts_at":      reservation.StartsAt,
		"ends_at":        reservation.EndsAt,
		"released_at":    reservation.ReleasedAt,
		"cancelled_at":   reservation.CancelledAt,
		"release_reason": nullableString(reservation.ReleaseReason),
	}

	boundQuery, arguments, err := sqlx.Named(insertQuery, params)
	if err != nil {
		return domain.Reservation{}, fmt.Errorf("build create reservation query: %w", err)
	}

	reboundQuery := repository.database.Rebind(boundQuery)

	var row reservationRow
	if err := repository.database.GetContext(ctx, &row, reboundQuery, arguments...); err != nil {
		if isReservationConflict(err) {
			return domain.Reservation{}, domain.ErrConflict
		}

		return domain.Reservation{}, fmt.Errorf("create reservation: %w", err)
	}

	return row.toDomain(), nil
}

// GetByID returns a reservation by identifier.
func (repository *ReservationRepository) GetByID(ctx context.Context, reservationID string) (domain.Reservation, error) {
	const query = `
		SELECT
			id,
			desk_id,
			user_id,
			source,
			status,
			holder_name,
			note,
			starts_at,
			ends_at,
			released_at,
			cancelled_at,
			release_reason,
			created_at,
			updated_at
		FROM reservations
		WHERE id = $1
	`

	var row reservationRow
	if err := repository.database.GetContext(ctx, &row, query, reservationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Reservation{}, domain.ErrNotFound
		}

		return domain.Reservation{}, fmt.Errorf("get reservation by id: %w", err)
	}

	return row.toDomain(), nil
}

// List returns reservations filtered by repository filter fields.
func (repository *ReservationRepository) List(ctx context.Context, filter repository.ReservationFilter) ([]domain.Reservation, error) {
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT
			id,
			desk_id,
			user_id,
			source,
			status,
			holder_name,
			note,
			starts_at,
			ends_at,
			released_at,
			cancelled_at,
			release_reason,
			created_at,
			updated_at
		FROM reservations
	`)

	conditions := make([]string, 0, 5)
	arguments := make([]any, 0, 5)
	argumentIndex := 1

	if filter.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argumentIndex))
		arguments = append(arguments, filter.UserID)
		argumentIndex++
	}

	if filter.DeskID != "" {
		conditions = append(conditions, fmt.Sprintf("desk_id = $%d", argumentIndex))
		arguments = append(arguments, filter.DeskID)
		argumentIndex++
	}

	if len(filter.DeskIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("desk_id = ANY($%d)", argumentIndex))
		arguments = append(arguments, pq.Array(filter.DeskIDs))
		argumentIndex++
	}

	if filter.FloorID != "" {
		conditions = append(conditions, fmt.Sprintf("desk_id IN (SELECT id FROM desks WHERE floor_id = $%d)", argumentIndex))
		arguments = append(arguments, filter.FloorID)
		argumentIndex++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argumentIndex))
		arguments = append(arguments, string(filter.Status))
		argumentIndex++
	}

	if filter.StartsAt != nil {
		conditions = append(conditions, fmt.Sprintf("ends_at > $%d", argumentIndex))
		arguments = append(arguments, *filter.StartsAt)
		argumentIndex++
	}

	if filter.EndsAt != nil {
		conditions = append(conditions, fmt.Sprintf("starts_at < $%d", argumentIndex))
		arguments = append(arguments, *filter.EndsAt)
		argumentIndex++
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}

	queryBuilder.WriteString(" ORDER BY starts_at ASC, id ASC")

	rows := make([]reservationRow, 0)
	if err := repository.database.SelectContext(ctx, &rows, queryBuilder.String(), arguments...); err != nil {
		return nil, fmt.Errorf("list reservations: %w", err)
	}

	reservations := make([]domain.Reservation, 0, len(rows))
	for _, row := range rows {
		reservations = append(reservations, row.toDomain())
	}

	return reservations, nil
}

// Cancel marks an active reservation as cancelled.
func (repository *ReservationRepository) Cancel(ctx context.Context, reservationID string, cancelledAt time.Time) error {
	logctx.Logger(ctx).Info("cancel reservation", "reservationId", reservationID)

	const query = `
		UPDATE reservations
		SET
			status = 'cancelled',
			cancelled_at = $2,
			updated_at = $2
		WHERE id = $1 AND status = 'active'
	`

	result, err := repository.database.ExecContext(ctx, query, reservationID, cancelledAt)
	if err != nil {
		return fmt.Errorf("cancel reservation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read cancel reservation result: %w", err)
	}

	if rowsAffected == 0 {
		return repository.resolveUpdateError(ctx, reservationID, "cancel")
	}

	return nil
}

// Release marks an active reservation as completed early.
func (repository *ReservationRepository) Release(ctx context.Context, reservationID string, releasedAt time.Time, reason string) error {
	logctx.Logger(ctx).Info("release reservation", "reservationId", reservationID)

	const query = `
		UPDATE reservations
		SET
			status = 'completed',
			released_at = $2,
			release_reason = $3,
			updated_at = $2
		WHERE id = $1 AND status = 'active'
	`

	result, err := repository.database.ExecContext(ctx, query, reservationID, releasedAt, nullableString(reason))
	if err != nil {
		return fmt.Errorf("release reservation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read release reservation result: %w", err)
	}

	if rowsAffected == 0 {
		return repository.resolveUpdateError(ctx, reservationID, "release")
	}

	return nil
}

// CompleteExpired transitions all active reservations whose end time is before now to completed.
func (repository *ReservationRepository) CompleteExpired(ctx context.Context, now time.Time) (int64, error) {
	const query = `
		UPDATE reservations
		SET status = 'completed', updated_at = $1
		WHERE status = 'active' AND ends_at < $1
	`

	result, err := repository.database.ExecContext(ctx, query, now)
	if err != nil {
		return 0, fmt.Errorf("complete expired reservations: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read complete expired result: %w", err)
	}

	return count, nil
}

// resolveUpdateError distinguishes between "reservation not found" and "reservation not in active status"
// when an UPDATE with WHERE status = 'active' affected zero rows.
func (repository *ReservationRepository) resolveUpdateError(ctx context.Context, reservationID string, operation string) error {
	const existsQuery = `SELECT status FROM reservations WHERE id = $1`

	var status string
	err := repository.database.GetContext(ctx, &status, existsQuery, reservationID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check reservation existence for %s: %w", operation, err)
	}

	return fmt.Errorf("reservation is already %s: %w", status, domain.ErrConflict)
}

func (row reservationRow) toDomain() domain.Reservation {
	reservation := domain.Reservation{
		ID:        row.ID,
		DeskID:    row.DeskID,
		UserID:    row.UserID,
		Source:    domain.ReservationSource(row.Source),
		Status:    domain.ReservationStatus(row.Status),
		StartsAt:  row.StartsAt,
		EndsAt:    row.EndsAt,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if row.HolderName.Valid {
		reservation.HolderName = row.HolderName.String
	}

	if row.Note.Valid {
		reservation.Note = row.Note.String
	}

	if row.ReleasedAt.Valid {
		releasedAt := row.ReleasedAt.Time
		reservation.ReleasedAt = &releasedAt
	}

	if row.CancelledAt.Valid {
		cancelledAt := row.CancelledAt.Time
		reservation.CancelledAt = &cancelledAt
	}

	if row.ReleaseReason.Valid {
		reservation.ReleaseReason = row.ReleaseReason.String
	}

	return reservation
}

func nullableString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func isReservationConflict(err error) bool {
	var postgresError *pq.Error
	if !errors.As(err, &postgresError) {
		return false
	}

	return postgresError.Code == "23P01"
}
