-- Up
-- Add qty_on_hold column to storages table
ALTER TABLE storages ADD COLUMN qty_on_hold INTEGER DEFAULT 0 NOT NULL;

-- Create qc_holds table
CREATE TABLE qc_holds (
    id VARCHAR(36) PRIMARY KEY,
    storage_id VARCHAR(36) NOT NULL,
    qty INTEGER NOT NULL,
    reason VARCHAR(50) NOT NULL,    -- DAMAGED, INVESTIGATION, EXPIRED, OTHER
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',  -- ACTIVE, RELEASED
    notes TEXT,
    created_by VARCHAR(36) NOT NULL,
    released_by VARCHAR(36),
    created_at TIMESTAMP NOT NULL,
    released_at TIMESTAMP,
    FOREIGN KEY (storage_id) REFERENCES storages(id)
);

CREATE INDEX idx_qc_holds_storage_id ON qc_holds(storage_id);
CREATE INDEX idx_qc_holds_status ON qc_holds(status);

-- Down
-- DROP TABLE qc_holds;
-- ALTER TABLE storages DROP COLUMN qty_on_hold;
