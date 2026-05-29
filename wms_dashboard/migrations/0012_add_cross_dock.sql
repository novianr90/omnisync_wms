-- Add IsCrossDock flag to inventory_movements
ALTER TABLE inventory_movements ADD COLUMN is_cross_dock BOOLEAN NOT NULL DEFAULT 0;

-- Seed the CROSS-DOCK staging area locator
INSERT INTO locators (id, zone, aisle, rack, level, barcode, is_active) 
VALUES ('loc-crossdock-01', 'ZONE-CROSSDOCK', 'CD-1', '01', '1', 'CD-1-01-1', 1)
ON CONFLICT(id) DO NOTHING;
