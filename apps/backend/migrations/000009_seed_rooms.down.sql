-- Remove seed: base floors, zones, desks and desk features
-- Deleting floors cascades to zones, desks and desk_features automatically.

DELETE FROM floors
WHERE id IN (
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000102'
);
