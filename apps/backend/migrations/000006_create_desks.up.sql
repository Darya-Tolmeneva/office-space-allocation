CREATE TABLE desks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    floor_id UUID NOT NULL REFERENCES floors(id) ON DELETE CASCADE,
    zone_id UUID NULL REFERENCES zones(id) ON DELETE SET NULL,
    label TEXT NOT NULL,
    state TEXT NOT NULL,
    position_x DOUBLE PRECISION NOT NULL,
    position_y DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT desks_state_check CHECK (state IN ('active', 'disabled')),
    CONSTRAINT desks_position_x_check CHECK (position_x >= 0 AND position_x <= 100),
    CONSTRAINT desks_position_y_check CHECK (position_y >= 0 AND position_y <= 100)
);

CREATE UNIQUE INDEX desks_floor_id_label_uidx ON desks (floor_id, label);
CREATE INDEX desks_floor_id_idx ON desks (floor_id);
CREATE INDEX desks_zone_id_idx ON desks (zone_id);
