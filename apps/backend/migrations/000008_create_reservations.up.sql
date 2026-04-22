CREATE TABLE reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    desk_id UUID NOT NULL REFERENCES desks(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    source TEXT NOT NULL,
    status TEXT NOT NULL,
    holder_name TEXT NULL,
    note TEXT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    released_at TIMESTAMPTZ NULL,
    cancelled_at TIMESTAMPTZ NULL,
    release_reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT reservations_source_check CHECK (source IN ('manual', 'auto')),
    CONSTRAINT reservations_status_check CHECK (status IN ('active', 'cancelled', 'completed')),
    CONSTRAINT reservations_time_range_check CHECK (starts_at < ends_at),
    CONSTRAINT reservations_cancelled_at_status_check CHECK (
        (cancelled_at IS NULL) OR (status = 'cancelled')
    ),
    CONSTRAINT reservations_released_at_status_check CHECK (
        (released_at IS NULL) OR (status = 'completed')
    )
);

ALTER TABLE reservations
    ADD CONSTRAINT reservations_active_desk_time_excl
    EXCLUDE USING GIST (
        desk_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    )
    WHERE (status = 'active');

CREATE INDEX reservations_user_id_created_at_idx ON reservations (user_id, created_at DESC);
CREATE INDEX reservations_desk_id_idx ON reservations (desk_id);
CREATE INDEX reservations_status_starts_at_idx ON reservations (status, starts_at);
CREATE INDEX reservations_starts_at_ends_at_idx ON reservations (starts_at, ends_at);
CREATE INDEX reservations_created_at_idx ON reservations (created_at);
CREATE INDEX reservations_active_desk_time_idx ON reservations (desk_id, starts_at, ends_at) WHERE status = 'active';
