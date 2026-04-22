package domain

import "time"

// Floor represents a physical office floor in the single-office setup.
type Floor struct {
	ID           string
	Name         string
	Timezone     string
	FloorPlanURL string
	CreatedAt    time.Time
}
