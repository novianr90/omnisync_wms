-- Up
CREATE TABLE inventory_kittings (
    id VARCHAR(36) PRIMARY KEY,
    document_no VARCHAR(50) NOT NULL UNIQUE,
    status VARCHAR(20) DEFAULT 'OPEN',
    finished_product_id VARCHAR(36) NOT NULL,
    finished_locator_id VARCHAR(36) NOT NULL,
    finished_qty INTEGER NOT NULL,
    remarks TEXT,
    created_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (finished_product_id) REFERENCES products(id),
    FOREIGN KEY (finished_locator_id) REFERENCES locators(id)
);

CREATE TABLE inventory_kitting_lines (
    id VARCHAR(36) PRIMARY KEY,
    kitting_id VARCHAR(36) NOT NULL,
    product_id VARCHAR(36) NOT NULL,
    locator_id VARCHAR(36) NOT NULL,
    consumed_qty INTEGER NOT NULL,
    FOREIGN KEY (kitting_id) REFERENCES inventory_kittings(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (locator_id) REFERENCES locators(id)
);

CREATE INDEX idx_inv_kit_lines_kit_id ON inventory_kitting_lines(kitting_id);
CREATE INDEX idx_inv_kit_lines_prod_id ON inventory_kitting_lines(product_id);
CREATE INDEX idx_inv_kit_lines_loc_id ON inventory_kitting_lines(locator_id);

-- Down
-- DROP TABLE inventory_kitting_lines;
-- DROP TABLE inventory_kittings;
