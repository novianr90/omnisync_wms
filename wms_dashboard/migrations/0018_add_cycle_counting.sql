-- 0018_add_cycle_counting.sql

ALTER TABLE locators ADD COLUMN is_frozen BOOLEAN DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS cycle_counts (
    id VARCHAR(36) PRIMARY KEY,
    document_no VARCHAR(50) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'CREATED',
    remarks TEXT,
    created_by VARCHAR(36) NOT NULL,
    adjusted_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
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

DROP VIEW IF EXISTS in_progress_documents;
CREATE VIEW in_progress_documents AS
SELECT 
    id, 
    document_no, 
    'Movement (' || movement_type || ')' AS doc_type, 
    created_at, 
    status, 
    '/wms/movements/' || id AS link 
FROM inventory_movements 
WHERE status NOT IN ('COMPLETED', 'REJECTED')

UNION ALL

SELECT 
    id, 
    document_no, 
    'QC Hold' AS doc_type, 
    created_at, 
    status, 
    '/wms/qc-holds' AS link 
FROM qc_holds 
WHERE status = 'ACTIVE'

UNION ALL

SELECT 
    id, 
    document_no, 
    'Adjustment' AS doc_type, 
    created_at, 
    status, 
    '/wms/adjustments' AS link 
FROM inventory_adjustments 
WHERE status = 'OPEN'

UNION ALL

SELECT 
    id, 
    document_no, 
    'Kitting' AS doc_type, 
    created_at, 
    status, 
    '/wms/kitting' AS link 
FROM inventory_kittings 
WHERE status = 'OPEN'

UNION ALL

SELECT
    id,
    document_no,
    'Cycle Count' AS doc_type,
    created_at,
    status,
    '/wms/cycle-counts/' || id AS link
FROM cycle_counts
WHERE status NOT IN ('COMPLETED', 'CANCELED');
