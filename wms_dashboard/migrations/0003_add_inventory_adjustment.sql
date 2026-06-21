-- Up
CREATE TABLE inventory_adjustments (
    id VARCHAR(36) PRIMARY KEY,
    document_no VARCHAR(50) NOT NULL UNIQUE,
    status VARCHAR(20) DEFAULT 'OPEN',
    reason_code VARCHAR(50) NOT NULL,
    remarks TEXT,
    created_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE TABLE inventory_adjustment_lines (
    id VARCHAR(36) PRIMARY KEY,
    adjustment_id VARCHAR(36) NOT NULL,
    product_id VARCHAR(36) NOT NULL,
    locator_id VARCHAR(36) NOT NULL,
    qty_delta INTEGER NOT NULL,
    FOREIGN KEY (adjustment_id) REFERENCES inventory_adjustments(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (locator_id) REFERENCES locators(id)
);

CREATE INDEX idx_inv_adj_lines_adj_id ON inventory_adjustment_lines(adjustment_id);
CREATE INDEX idx_inv_adj_lines_prod_id ON inventory_adjustment_lines(product_id);
CREATE INDEX idx_inv_adj_lines_loc_id ON inventory_adjustment_lines(locator_id);

-- Down
-- DROP TABLE inventory_adjustment_lines;
-- DROP TABLE inventory_adjustments;
