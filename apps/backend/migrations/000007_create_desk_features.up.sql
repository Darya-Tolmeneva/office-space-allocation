CREATE TABLE desk_features (
    desk_id UUID NOT NULL REFERENCES desks(id) ON DELETE CASCADE,
    feature TEXT NOT NULL,
    PRIMARY KEY (desk_id, feature),
    CONSTRAINT desk_features_feature_check CHECK (feature IN (
        'monitor',
        'dual_monitor',
        'wifi',
        'ethernet',
        'standing',
        'quiet',
        'window',
        'near_kitchen',
        'accessible'
    ))
);

CREATE INDEX desk_features_feature_desk_id_idx ON desk_features (feature, desk_id);
