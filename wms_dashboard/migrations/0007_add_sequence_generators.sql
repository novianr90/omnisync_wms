-- Up
CREATE TABLE sequence_generators (
    id VARCHAR(36) PRIMARY KEY,
    usage_table VARCHAR(50) UNIQUE NOT NULL,
    prefix VARCHAR(10) NOT NULL,
    fiscal_year INTEGER NOT NULL,
    current_number INTEGER NOT NULL DEFAULT 1,
    number_length INTEGER NOT NULL DEFAULT 5,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO sequence_generators (id, usage_table, prefix, fiscal_year, current_number, number_length) VALUES
('seq-mov', 'inventory_movements', 'MOV', 2026, 1, 5),
('seq-adj', 'inventory_adjustments', 'ADJ', 2026, 12, 5),
('seq-kit', 'inventory_kittings', 'KIT', 2026, 3, 5),
('seq-qch', 'qc_holds', 'QCH', 2026, 1, 5),
('seq-stor', 'storages', 'BAT', 2026, 125, 6);

ALTER TABLE qc_holds ADD COLUMN document_no VARCHAR(50);
CREATE UNIQUE INDEX idx_qc_holds_document_no ON qc_holds(document_no);

-- Down
-- DROP INDEX idx_qc_holds_document_no;
-- ALTER TABLE qc_holds DROP COLUMN document_no;
-- DROP TABLE sequence_generators;
