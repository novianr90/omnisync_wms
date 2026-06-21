-- 0018_add_cycle_counting.sql

ALTER TABLE locators ADD COLUMN is_frozen BOOLEAN DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS cycle_counts (
    id VARCHAR(36) PRIMARY KEY,
    document_no VARCHAR(50) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'DRAFT',
    remarks TEXT,
    created_by VARCHAR(36) NOT NULL,
    adjusted_at DATETIME,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS cycle_count_lines (
    id VARCHAR(36) PRIMARY KEY,
    cycle_count_id VARCHAR(36) NOT NULL,
    locator_id VARCHAR(36) NOT NULL,
    product_id VARCHAR(36) NOT NULL,
    expected_qty INTEGER NOT NULL,
    counted_qty INTEGER,
    variance INTEGER DEFAULT 0,
    is_frozen BOOLEAN DEFAULT FALSE,
    FOREIGN KEY(cycle_count_id) REFERENCES cycle_counts(id),
    FOREIGN KEY(locator_id) REFERENCES locators(id),
    FOREIGN KEY(product_id) REFERENCES products(id)
);

CREATE INDEX IF NOT EXISTS idx_cycle_count_lines_ccid ON cycle_count_lines(cycle_count_id);
CREATE INDEX IF NOT EXISTS idx_cycle_count_lines_loc ON cycle_count_lines(locator_id);
CREATE INDEX IF NOT EXISTS idx_cycle_count_lines_prod ON cycle_count_lines(product_id);

INSERT INTO sequence_generators (id, usage_table, prefix, fiscal_year, current_number, number_length, created_at, updated_at) 
VALUES ('seq_cycle_counts', 'cycle_counts', 'CNT', 2026, 1, 5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
