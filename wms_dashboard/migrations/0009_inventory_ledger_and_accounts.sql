CREATE TABLE IF NOT EXISTS accounts (
    account_no VARCHAR(50) PRIMARY KEY,
    account_name VARCHAR(100) NOT NULL,
    account_type VARCHAR(50) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS inventory_ledgers (
    id VARCHAR(36) PRIMARY KEY,
    transaction_date DATETIME NOT NULL,
    product_id VARCHAR(36) NOT NULL,
    locator_id VARCHAR(36) NOT NULL,
    batch_number VARCHAR(100) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    document_no VARCHAR(50) NOT NULL,
    qty_change INTEGER NOT NULL,
    batch_balance INTEGER NOT NULL,
    account_no VARCHAR(50),
    contra_account_no VARCHAR(50),
    created_by VARCHAR(36) NOT NULL,
    
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (locator_id) REFERENCES locators(id),
    FOREIGN KEY (account_no) REFERENCES accounts(account_no),
    FOREIGN KEY (contra_account_no) REFERENCES accounts(account_no)
);

CREATE INDEX idx_inventory_ledgers_transaction_date ON inventory_ledgers(transaction_date);
CREATE INDEX idx_inventory_ledgers_product_id ON inventory_ledgers(product_id);
CREATE INDEX idx_inventory_ledgers_locator_id ON inventory_ledgers(locator_id);
CREATE INDEX idx_inventory_ledgers_batch_number ON inventory_ledgers(batch_number);
CREATE INDEX idx_inventory_ledgers_transaction_type ON inventory_ledgers(transaction_type);
CREATE INDEX idx_inventory_ledgers_document_no ON inventory_ledgers(document_no);
