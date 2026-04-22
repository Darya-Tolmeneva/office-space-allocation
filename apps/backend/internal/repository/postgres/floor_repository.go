package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"office-space-allocation/apps/backend/internal/domain"
)

// FloorRepository provides PostgreSQL access to floors.
type FloorRepository struct {
	database *sqlx.DB
}

// NewFloorRepository creates a floor repository backed by PostgreSQL.
func NewFloorRepository(database *sqlx.DB) *FloorRepository {
	return &FloorRepository{database: database}
}

type floorRow struct {
	ID           string    `db:"id"`
	Name         string    `db:"name"`
	Timezone     string    `db:"timezone"`
	FloorPlanURL *string   `db:"floor_plan_url"`
	CreatedAt    time.Time `db:"created_at"`
}

// List returns all floors ordered by name.
func (repository *FloorRepository) List(ctx context.Context) ([]domain.Floor, error) {
	const query = `
		SELECT id, name, timezone, floor_plan_url, created_at
		FROM floors
		ORDER BY name ASC, id ASC
	`

	rows := make([]floorRow, 0)
	if err := repository.database.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("list floors: %w", err)
	}

	floors := make([]domain.Floor, 0, len(rows))
	for _, row := range rows {
		floors = append(floors, row.toDomain())
	}

	return floors, nil
}

// GetByID returns a floor by identifier.
func (repository *FloorRepository) GetByID(ctx context.Context, floorID string) (domain.Floor, error) {
	const query = `
		SELECT id, name, timezone, floor_plan_url, created_at
		FROM floors
		WHERE id = $1
	`

	var row floorRow
	if err := repository.database.GetContext(ctx, &row, query, floorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Floor{}, domain.ErrNotFound
		}

		return domain.Floor{}, fmt.Errorf("get floor by id: %w", err)
	}

	return row.toDomain(), nil
}

func (row floorRow) toDomain() domain.Floor {
	floor := domain.Floor{
		ID:        row.ID,
		Name:      row.Name,
		Timezone:  row.Timezone,
		CreatedAt: row.CreatedAt,
	}

	if row.FloorPlanURL != nil {
		floor.FloorPlanURL = *row.FloorPlanURL
	}

	return floor
}
