CREATE TABLE zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    floor_id UUID NOT NULL REFERENCES floors(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT zones_type_check CHECK (type IN ('open_space', 'meeting_room', 'phone_booth', 'quiet_zone'))
);

CREATE UNIQUE INDEX zones_floor_id_name_uidx ON zones (floor_id, name);
CREATE INDEX zones_floor_id_idx ON zones (floor_id);
