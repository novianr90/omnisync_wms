-- 0019_update_in_progress_view.sql

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
