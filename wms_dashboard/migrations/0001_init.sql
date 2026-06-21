CREATE TABLE IF NOT EXISTS uoms (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(36) PRIMARY KEY,
    sku VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    price DECIMAL(12,2) DEFAULT 0.00,
    uom_id VARCHAR(36),
    created_at TIMESTAMP,
    deleted_at TIMESTAMP,
    FOREIGN KEY (uom_id) REFERENCES uoms(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS uom_conversions (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36),
    from_uom_id VARCHAR(36) NOT NULL,
    to_uom_id VARCHAR(36) NOT NULL,
    multiply_factor DECIMAL(12,6) DEFAULT 1.0,
    created_at TIMESTAMP,
    deleted_at TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (from_uom_id) REFERENCES uoms(id) ON DELETE CASCADE,
    FOREIGN KEY (to_uom_id) REFERENCES uoms(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS warehouses (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    address TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS locators (
    id VARCHAR(36) PRIMARY KEY,
    warehouse_id VARCHAR(36) NOT NULL,
    zone VARCHAR(20) NOT NULL,
    aisle VARCHAR(20) NOT NULL,
    shelf VARCHAR(20) NOT NULL,
    level VARCHAR(20) NOT NULL,
    code VARCHAR(100) NOT NULL UNIQUE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP,
    deleted_at TIMESTAMP,
    FOREIGN KEY (warehouse_id) REFERENCES warehouses(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS storages (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36) NOT NULL,
    locator_id VARCHAR(36) NOT NULL,
    batch_number VARCHAR(100) NOT NULL,
    serial_number VARCHAR(100),
    received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    qty_on_hand INTEGER DEFAULT 0,
    qty_reserved INTEGER DEFAULT 0,
    updated_at TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (locator_id) REFERENCES locators(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS inventory_movements (
    id VARCHAR(36) PRIMARY KEY,
    document_no VARCHAR(50) NOT NULL UNIQUE,
    movement_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'OPEN',
    created_by VARCHAR(36) NOT NULL,
    assigned_operator_id VARCHAR(36),
    remarks TEXT,
    rejection_reason TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS inventory_movement_lines (
    id VARCHAR(36) PRIMARY KEY,
    movement_id VARCHAR(36) NOT NULL,
    product_id VARCHAR(36) NOT NULL,
    batch_number VARCHAR(100),
    from_locator_id VARCHAR(36),
    to_locator_id VARCHAR(36),
    requested_quantity INTEGER NOT NULL,
    actual_quantity INTEGER DEFAULT 0,
    FOREIGN KEY (movement_id) REFERENCES inventory_movements(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (from_locator_id) REFERENCES locators(id) ON DELETE SET NULL,
    FOREIGN KEY (to_locator_id) REFERENCES locators(id) ON DELETE SET NULL
);
