-- Up
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id VARCHAR(36) NOT NULL,
    permission VARCHAR(100) NOT NULL,
    PRIMARY KEY (role_id, permission),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

INSERT INTO role_permissions (role_id, permission) VALUES
('role-sys-admin-001', 'view_ledger'),
('role-sys-admin-001', 'modify_masters'),
('role-sys-admin-001', 'manage_system'),
('role-admin-wms-002', 'modify_masters'),
('role-admin-wms-002', 'manage_system');

-- Down
-- DROP TABLE role_permissions;
