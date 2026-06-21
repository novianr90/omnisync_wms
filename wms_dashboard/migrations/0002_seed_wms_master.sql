-- 0. Seed UoMs
INSERT INTO uoms (id, code, name, description, created_at) VALUES 
('uom-kg-0001', 'kg', 'Kilogram', 'Standard metric unit for weight', CURRENT_TIMESTAMP),
('uom-pack-002', 'pack', 'Pack', 'Pack unit of items', CURRENT_TIMESTAMP),
('uom-box-0003', 'box', 'Box', 'Box containing multiple individual items', CURRENT_TIMESTAMP),
('uom-pcs-0004', 'pcs', 'Pieces', 'Individual singular units', CURRENT_TIMESTAMP);

-- 1. Seed Warehouses
INSERT INTO warehouses (id, code, name, address, is_active, created_at) VALUES 
('wh-main-0001', 'WH-MAIN', 'Central Logistics Hub', 'Golden Gate Sector 4, Silicon Valley', TRUE, CURRENT_TIMESTAMP);

-- 2. Seed Locators (Shelves)
INSERT INTO locators (id, warehouse_id, zone, aisle, shelf, level, code, is_active, created_at) VALUES 
('loc-001', 'wh-main-0001', 'Zone-A', 'Aisle-1', 'Shelf-1', 'Level-1', 'WH-MAIN-Zone-A-Aisle-1-Shelf-1-Level-1', TRUE, CURRENT_TIMESTAMP),
('loc-002', 'wh-main-0001', 'Zone-A', 'Aisle-1', 'Shelf-1', 'Level-2', 'WH-MAIN-Zone-A-Aisle-1-Shelf-1-Level-2', TRUE, CURRENT_TIMESTAMP),
('loc-003', 'wh-main-0001', 'Zone-A', 'Aisle-1', 'Shelf-2', 'Level-1', 'WH-MAIN-Zone-A-Aisle-1-Shelf-2-Level-1', TRUE, CURRENT_TIMESTAMP),
('loc-004', 'wh-main-0001', 'Zone-A', 'Aisle-1', 'Shelf-2', 'Level-2', 'WH-MAIN-Zone-A-Aisle-1-Shelf-2-Level-2', TRUE, CURRENT_TIMESTAMP),
('loc-005', 'wh-main-0001', 'Zone-A', 'Aisle-2', 'Shelf-1', 'Level-1', 'WH-MAIN-Zone-A-Aisle-2-Shelf-1-Level-1', TRUE, CURRENT_TIMESTAMP),
('loc-006', 'wh-main-0001', 'Zone-A', 'Aisle-2', 'Shelf-1', 'Level-2', 'WH-MAIN-Zone-A-Aisle-2-Shelf-1-Level-2', TRUE, CURRENT_TIMESTAMP),
('loc-007', 'wh-main-0001', 'Zone-A', 'Aisle-2', 'Shelf-2', 'Level-1', 'WH-MAIN-Zone-A-Aisle-2-Shelf-2-Level-1', TRUE, CURRENT_TIMESTAMP),
('loc-008', 'wh-main-0001', 'Zone-A', 'Aisle-2', 'Shelf-2', 'Level-2', 'WH-MAIN-Zone-A-Aisle-2-Shelf-2-Level-2', TRUE, CURRENT_TIMESTAMP),
('loc-009', 'wh-main-0001', 'Zone-B', 'Aisle-1', 'Shelf-1', 'Level-1', 'WH-MAIN-Zone-B-Aisle-1-Shelf-1-Level-1', TRUE, CURRENT_TIMESTAMP),
('loc-010', 'wh-main-0001', 'Zone-B', 'Aisle-1', 'Shelf-1', 'Level-2', 'WH-MAIN-Zone-B-Aisle-1-Shelf-1-Level-2', TRUE, CURRENT_TIMESTAMP),
('loc-011', 'wh-main-0001', 'Zone-B', 'Aisle-1', 'Shelf-2', 'Level-1', 'WH-MAIN-Zone-B-Aisle-1-Shelf-2-Level-1', TRUE, CURRENT_TIMESTAMP),
('loc-012', 'wh-main-0001', 'Zone-B', 'Aisle-1', 'Shelf-2', 'Level-2', 'WH-MAIN-Zone-B-Aisle-1-Shelf-2-Level-2', TRUE, CURRENT_TIMESTAMP),
('loc-013', 'wh-main-0001', 'Zone-B', 'Aisle-2', 'Shelf-1', 'Level-1', 'WH-MAIN-Zone-B-Aisle-2-Shelf-1-Level-1', TRUE, CURRENT_TIMESTAMP),
('loc-014', 'wh-main-0001', 'Zone-B', 'Aisle-2', 'Shelf-1', 'Level-2', 'WH-MAIN-Zone-B-Aisle-2-Shelf-1-Level-2', TRUE, CURRENT_TIMESTAMP),
('loc-015', 'wh-main-0001', 'Zone-B', 'Aisle-2', 'Shelf-2', 'Level-1', 'WH-MAIN-Zone-B-Aisle-2-Shelf-2-Level-1', TRUE, CURRENT_TIMESTAMP),
('loc-016', 'wh-main-0001', 'Zone-B', 'Aisle-2', 'Shelf-2', 'Level-2', 'WH-MAIN-Zone-B-Aisle-2-Shelf-2-Level-2', TRUE, CURRENT_TIMESTAMP);

-- 3. Seed Products
INSERT INTO products (id, sku, name, description, category, price, uom_id, created_at) VALUES 
('prod-001', 'PROD-KYBD-01', 'Mechanical Keychron K2 Keyboard', 'Wireless 84-Key mechanical keyboard with Gateron switches', 'Electronics', 89.99, 'uom-pcs-0004', CURRENT_TIMESTAMP),
('prod-002', 'PROD-MOUS-02', 'Logitech MX Master 3S Mouse', 'Ergonomic wireless office mouse with silent clicks', 'Electronics', 99.99, 'uom-pcs-0004', CURRENT_TIMESTAMP),
('prod-003', 'PROD-MON-03', 'Dell UltraSharp 27" 4K Monitor', 'U2723QE USB-C Hub monitor with IPS Black technology', 'Electronics', 499.00, 'uom-pcs-0004', CURRENT_TIMESTAMP),
('prod-004', 'PROD-SUGR-04', 'Refined White Sugar', 'Fine granular white table sugar', 'Consumables', 1.99, 'uom-kg-0001', CURRENT_TIMESTAMP);

-- 3.5 Seed Sugar Conversion Rule (1 pack of sugar = 1.0 kg of sugar)
INSERT INTO uom_conversions (id, product_id, from_uom_id, to_uom_id, multiply_factor, created_at) VALUES 
('conv-001', 'prod-004', 'uom-pack-002', 'uom-kg-0001', 1.0, CURRENT_TIMESTAMP);

-- 4. Seed Storage Lots (with FIFO demo items!)
INSERT INTO storages (id, product_id, locator_id, batch_number, received_at, qty_on_hand, qty_reserved, updated_at) VALUES 
-- Batch 1 (Oldest - Received 5 days ago)
('stor-001', 'prod-001', 'loc-001', 'BAT-INB-20260520-1', '2026-05-20 00:00:00', 40, 0, CURRENT_TIMESTAMP),
-- Batch 2 (Newer - Received today)
('stor-002', 'prod-001', 'loc-001', 'BAT-INB-20260525-2', CURRENT_TIMESTAMP, 60, 0, CURRENT_TIMESTAMP),
-- Mouse Storage (Received 2 days ago)
('stor-003', 'prod-002', 'loc-002', 'BAT-INB-20260523-1', '2026-05-23 00:00:00', 80, 0, CURRENT_TIMESTAMP),
-- Sugar Storage (Received 1 day ago)
('stor-004', 'prod-004', 'loc-001', 'BAT-SUGR-20260524-1', '2026-05-24 00:00:00', 150, 0, CURRENT_TIMESTAMP);
