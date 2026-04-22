package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"office-space-allocation/apps/backend/internal/domain"
	repo "office-space-allocation/apps/backend/internal/repository"
)

// DeskRepository provides PostgreSQL access to desks.
type DeskRepository struct {
	database *sqlx.DB
}

// NewDeskRepository creates a desk repository backed by PostgreSQL.
func NewDeskRepository(database *sqlx.DB) *DeskRepository {
	return &DeskRepository{database: database}
}

type deskRow struct {
	ID        string         `db:"id"`
	FloorID   string         `db:"floor_id"`
	ZoneID    sql.NullString `db:"zone_id"`
	Label     string         `db:"label"`
	State     string         `db:"state"`
	PositionX float64        `db:"position_x"`
	PositionY float64        `db:"position_y"`
	CreatedAt time.Time      `db:"created_at"`
	Features  pq.StringArray `db:"features"`
}

// List returns desks filtered by floor, zone, and required features.
func (repository *DeskRepository) List(ctx context.Context, filter repo.DeskFilter) ([]domain.Desk, error) {
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT
			d.id,
			d.floor_id,
			d.zone_id,
			d.label,
			d.state,
			d.position_x,
			d.position_y,
			d.created_at,
			COALESCE(array_agg(df.feature ORDER BY df.feature) FILTER (WHERE df.feature IS NOT NULL), '{}') AS features
		FROM desks d
		LEFT JOIN desk_features df ON df.desk_id = d.id
	`)

	conditions := make([]string, 0, 2)
	arguments := make([]any, 0, 4)
	argumentIndex := 1

	if filter.FloorID != "" {
		conditions = append(conditions, fmt.Sprintf("d.floor_id = $%d", argumentIndex))
		arguments = append(arguments, filter.FloorID)
		argumentIndex++
	}

	if filter.ZoneID != "" {
		conditions = append(conditions, fmt.Sprintf("d.zone_id = $%d", argumentIndex))
		arguments = append(arguments, filter.ZoneID)
		argumentIndex++
	}

	if filter.ActiveOnly {
		conditions = append(conditions, "d.state = 'active'")
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}

	queryBuilder.WriteString(`
		GROUP BY d.id, d.floor_id, d.zone_id, d.label, d.state, d.position_x, d.position_y, d.created_at
	`)

	if len(filter.Features) > 0 {
		featureValues := make([]string, 0, len(filter.Features))
		for _, feature := range filter.Features {
			featureValues = append(featureValues, string(feature))
		}

		queryBuilder.WriteString(fmt.Sprintf(" HAVING COUNT(DISTINCT CASE WHEN df.feature = ANY($%d) THEN df.feature END) = $%d", argumentIndex, argumentIndex+1))
		arguments = append(arguments, pq.Array(featureValues), len(featureValues))
		argumentIndex += 2
	}

	queryBuilder.WriteString(" ORDER BY d.label ASC, d.id ASC")

	rows := make([]deskRow, 0)
	if err := repository.database.SelectContext(ctx, &rows, queryBuilder.String(), arguments...); err != nil {
		return nil, fmt.Errorf("list desks: %w", err)
	}

	desks := make([]domain.Desk, 0, len(rows))
	for _, row := range rows {
		desks = append(desks, row.toDomain())
	}

	return desks, nil
}

// ListByIDs returns desks for a given set of identifiers in a single query.
func (repository *DeskRepository) ListByIDs(ctx context.Context, deskIDs []string) ([]domain.Desk, error) {
	if len(deskIDs) == 0 {
		return nil, nil
	}

	const query = `
		SELECT
			d.id,
			d.floor_id,
			d.zone_id,
			d.label,
			d.state,
			d.position_x,
			d.position_y,
			d.created_at,
			COALESCE(array_agg(df.feature ORDER BY df.feature) FILTER (WHERE df.feature IS NOT NULL), '{}') AS features
		FROM desks d
		LEFT JOIN desk_features df ON df.desk_id = d.id
		WHERE d.id = ANY($1)
		GROUP BY d.id, d.floor_id, d.zone_id, d.label, d.state, d.position_x, d.position_y, d.created_at
		ORDER BY d.label ASC, d.id ASC
	`

	rows := make([]deskRow, 0, len(deskIDs))
	if err := repository.database.SelectContext(ctx, &rows, query, pq.Array(deskIDs)); err != nil {
		return nil, fmt.Errorf("list desks by ids: %w", err)
	}

	desks := make([]domain.Desk, 0, len(rows))
	for _, row := range rows {
		desks = append(desks, row.toDomain())
	}

	return desks, nil
}

// GetByID returns a desk by identifier.
func (repository *DeskRepository) GetByID(ctx context.Context, deskID string) (domain.Desk, error) {
	const query = `
		SELECT
			d.id,
			d.floor_id,
			d.zone_id,
			d.label,
			d.state,
			d.position_x,
			d.position_y,
			d.created_at,
			COALESCE(array_agg(df.feature ORDER BY df.feature) FILTER (WHERE df.feature IS NOT NULL), '{}') AS features
		FROM desks d
		LEFT JOIN desk_features df ON df.desk_id = d.id
		WHERE d.id = $1
		GROUP BY d.id, d.floor_id, d.zone_id, d.label, d.state, d.position_x, d.position_y, d.created_at
	`

	var row deskRow
	if err := repository.database.GetContext(ctx, &row, query, deskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Desk{}, domain.ErrNotFound
		}

		return domain.Desk{}, fmt.Errorf("get desk by id: %w", err)
	}

	return row.toDomain(), nil
}

// ListAvailability returns availability slots for a desk within the requested interval.
func (repository *DeskRepository) ListAvailability(ctx context.Context, deskID string, startsAt time.Time, endsAt time.Time) ([]repo.AvailabilitySlot, error) {
	desk, err := repository.GetByID(ctx, deskID)
	if err != nil {
		return nil, err
	}

	startsAt = startsAt.UTC()
	endsAt = endsAt.UTC()

	if desk.State != domain.DeskStateActive {
		return []repo.AvailabilitySlot{{
			From:   startsAt,
			To:     endsAt,
			Status: "disabled",
		}}, nil
	}

	const query = `
		SELECT
			id,
			starts_at,
			ends_at
		FROM reservations
		WHERE desk_id = $1
		  AND status = 'active'
		  AND starts_at < $3
		  AND ends_at > $2
		ORDER BY starts_at ASC, id ASC
	`

	type reservationIntervalRow struct {
		ID       string    `db:"id"`
		StartsAt time.Time `db:"starts_at"`
		EndsAt   time.Time `db:"ends_at"`
	}

	rows := make([]reservationIntervalRow, 0)
	if err := repository.database.SelectContext(ctx, &rows, query, deskID, startsAt, endsAt); err != nil {
		return nil, fmt.Errorf("list desk availability: %w", err)
	}

	type availabilityBoundary struct {
		At            time.Time
		ReservationID string
		Delta         int
	}

	boundaries := make([]availabilityBoundary, 0, len(rows)*2+2)
	boundaries = append(boundaries,
		availabilityBoundary{At: startsAt},
		availabilityBoundary{At: endsAt},
	)

	for _, row := range rows {
		from := maxTime(startsAt, row.StartsAt.UTC())
		to := minTime(endsAt, row.EndsAt.UTC())
		if !from.Before(to) {
			continue
		}

		boundaries = append(boundaries,
			availabilityBoundary{At: from, ReservationID: row.ID, Delta: 1},
			availabilityBoundary{At: to, ReservationID: row.ID, Delta: -1},
		)
	}

	sort.Slice(boundaries, func(i, j int) bool {
		if boundaries[i].At.Equal(boundaries[j].At) {
			return boundaries[i].Delta > boundaries[j].Delta
		}
		return boundaries[i].At.Before(boundaries[j].At)
	})

	activeReservations := make(map[string]int)
	slots := make([]repo.AvailabilitySlot, 0, len(boundaries))

	for index := 0; index < len(boundaries)-1; index++ {
		boundary := boundaries[index]
		if boundary.ReservationID != "" {
			if boundary.Delta > 0 {
				activeReservations[boundary.ReservationID] += boundary.Delta
			} else {
				delete(activeReservations, boundary.ReservationID)
			}
		}

		current := boundary.At.UTC()
		next := boundaries[index+1].At.UTC()
		if !current.Before(next) {
			continue
		}

		slot := repo.AvailabilitySlot{
			From: current,
			To:   next,
		}
		if len(activeReservations) == 0 {
			slot.Status = "available"
		} else {
			slot.Status = "reserved"
			for reservationID := range activeReservations {
				slot.ReservationID = reservationID
				break
			}
		}

		slots = append(slots, slot)
	}

	if len(slots) == 0 {
		return []repo.AvailabilitySlot{{
			From:   startsAt,
			To:     endsAt,
			Status: "available",
		}}, nil
	}

	return slots, nil
}

func minTime(left time.Time, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

func maxTime(left time.Time, right time.Time) time.Time {
	if left.After(right) {
		return left
	}
	return right
}

func (row deskRow) toDomain() domain.Desk {
	desk := domain.Desk{
		ID:      row.ID,
		FloorID: row.FloorID,
		Label:   row.Label,
		State:   domain.DeskState(row.State),
		Position: domain.DeskPosition{
			X: row.PositionX,
			Y: row.PositionY,
		},
		CreatedAt: row.CreatedAt,
		Features:  make([]domain.DeskFeature, 0, len(row.Features)),
	}

	if row.ZoneID.Valid {
		desk.ZoneID = row.ZoneID.String
	}

	for _, feature := range row.Features {
		desk.Features = append(desk.Features, domain.DeskFeature(feature))
	}

	return desk
}
