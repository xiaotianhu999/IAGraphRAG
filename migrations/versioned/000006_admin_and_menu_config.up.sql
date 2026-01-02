-- Migration: 000006_admin_and_menu_config
-- Description: Add menu_config to tenants and seed admin user

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS menu_config JSONB DEFAULT '["knowledge-bases", "creatChat", "settings"]';

-- Seed default system tenant if not exists
INSERT INTO tenants (id, name, description, api_key, status, business)
SELECT 10000, 'System Tenant', 'Default system tenant', 'system-api-key-' || uuid_generate_v4(), 'active', 'System'
WHERE NOT EXISTS (SELECT 1 FROM tenants WHERE id = 10000);

-- Seed admin user if not exists
-- Password is 'admin123' hashed with bcrypt
INSERT INTO users (id, username, email, password_hash, tenant_id, is_active, can_access_all_tenants)
SELECT uuid_generate_v4(), 'admin', 'admin@aiplusall.cn', '$2b$12$qxsyv90SdX9.q7jVbgO4P.GkM1BQJQHPTBnV1FdUzsu1R1ZqkoeAC', 10000, true, true
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = 'admin');
