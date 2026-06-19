-- Add capacity fields to locators and physical dimension fields to products
ALTER TABLE locators ADD COLUMN max_weight DECIMAL(10,2) DEFAULT 0;
ALTER TABLE locators ADD COLUMN max_volume DECIMAL(10,4) DEFAULT 0;
ALTER TABLE products ADD COLUMN unit_weight DECIMAL(10,4) DEFAULT 0;
ALTER TABLE products ADD COLUMN unit_volume DECIMAL(10,6) DEFAULT 0;
