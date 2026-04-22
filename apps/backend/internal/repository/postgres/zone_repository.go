package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"office-space-allocation/apps/backend/internal/domain"
)

// ZoneRepository provides PostgreSQL access to zones.
type ZoneRepository struct {
	database *sqlx.DB
}

// NewZoneRepository creates a zone repository backed by PostgreSQL.
func NewZoneRepository(database *sqlx.DB) *ZoneRepository {
	return &ZoneRepository{database: database}
}

type zoneRow struct {
	ID        string    `db:"id"`
	FloorID   string    `db:"floor_id"`
	Name      string    `db:"name"`
	Type      string    `db:"type"`
	CreatedAt time.Time `db:"created_at"`
}

// List returns all zones ordered by name.
func (repository *ZoneRepository) List(ctx context.Context) ([]domain.Zone, error) {
	const query = `
		SELECT id, floor_id, name, type, created_at
		FROM zones
		ORDER BY name ASC, id ASC
	`

	rows := make([]zoneRow, 0)
	if err := repository.database.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}

	zones := make([]domain.Zone, 0, len(rows))
	for _, row := range rows {
		zones = append(zones, row.toDomain())
	}

	return zones, nil
}

// ListByFloorID returns zones for a floor ordered by name.
func (repository *ZoneRepository) ListByFloorID(ctx context.Context, floorID string) ([]domain.Zone, error) {
	const query = `
		SELECT id, floor_id, name, type, created_at
		FROM zones
		WHERE floor_id = $1
		ORDER BY name ASC, id ASC
	`

	rows := make([]zoneRow, 0)
	if err := repository.database.SelectContext(ctx, &rows, query, floorID); err != nil {
		return nil, fmt.Errorf("list zones by floor id: %w", err)
	}

	zones := make([]domain.Zone, 0, len(rows))
	for _, row := range rows {
		zones = append(zones, row.toDomain())
	}

	return zones, nil
}

func (row zoneRow) toDomain() domain.Zone {
	return domain.Zone{
		ID:        row.ID,
		FloorID:   row.FloorID,
		Name:      row.Name,
		Type:      domain.ZoneType(row.Type),
		CreatedAt: row.CreatedAt,
	}
}
