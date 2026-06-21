-- Up
ALTER TABLE inventory_movement_lines ADD COLUMN is_from_hold BOOLEAN DEFAULT FALSE NOT NULL;

-- Down
-- ALTER TABLE inventory_movement_lines DROP COLUMN is_from_hold;
