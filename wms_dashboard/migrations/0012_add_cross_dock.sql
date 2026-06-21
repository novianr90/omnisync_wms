-- Add IsCrossDock flag to inventory_movements
ALTER TABLE inventory_movements ADD COLUMN is_cross_dock BOOLEAN NOT NULL DEFAULT FALSE;

-- Seed the CROSS-DOCK staging area locator
INSERT INTO locators (id, warehouse_id, zone, aisle, shelf, level, code, is_active) 
VALUES ('loc-crossdock-01', 'wh-main-0001', 'ZONE-CROSSDOCK', 'CD-1', '01', '1', 'CD-1-01-1', TRUE)
ON CONFLICT(id) DO NOTHING;
