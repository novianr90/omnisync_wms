CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role_id VARCHAR(36) NOT NULL,
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME,
    updated_at DATETIME,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE RESTRICT
);
