ALTER TABLE products ADD COLUMN is_bundle BOOLEAN DEFAULT FALSE;

INSERT INTO products (id, sku, name, description, category, price, is_bundle, uom_id, created_at) VALUES 
('prod-005', 'BDL-MS-KB', 'Logitech Mouse + Keychron Keyboard Bundle', 'Complete workspace input accessories bundle', 'Electronics', 179.99, TRUE, 'uom-pack-002', CURRENT_TIMESTAMP);

