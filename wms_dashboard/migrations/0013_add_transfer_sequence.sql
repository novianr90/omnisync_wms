-- Up
INSERT INTO sequence_generators (id, usage_table, prefix, fiscal_year, current_number, number_length) VALUES
('seq-trf', 'inventory_transfers', 'TRF', 2026, 1, 5);

-- Down
-- DELETE FROM sequence_generators WHERE id = 'seq-trf';
