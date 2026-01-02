-- Migration: 000009_user_menu_config
-- Description: Add menu_config column to users table for user-level menu permissions

DO $$ BEGIN RAISE NOTICE '[Migration 000009] Adding menu_config column to users table...'; END $$;

-- Add menu_config column to users table (use JSONB to match tenants table type)
ALTER TABLE users ADD COLUMN IF NOT EXISTS menu_config JSONB DEFAULT '[]'::jsonb;

COMMENT ON COLUMN users.menu_config IS 'Menu configuration for the user (overrides tenant default if set)';

-- For existing users, set menu_config based on their role:
-- Admin users: empty array (will inherit all permissions from role check)
-- Normal users: copy from tenant menu_config or set default ["creatChat"]
UPDATE users u
SET menu_config = CASE 
    WHEN EXISTS (SELECT 1 FROM tenants t WHERE t.id = u.tenant_id AND t.menu_config IS NOT NULL AND t.menu_config::text != '[]')
    THEN (SELECT t.menu_config FROM tenants t WHERE t.id = u.tenant_id)
    ELSE '["creatChat"]'::jsonb
END
WHERE u.role = 'user' 
  AND (u.menu_config IS NULL OR u.menu_config::text = '[]');

DO $$ BEGIN RAISE NOTICE '[Migration 000009] User menu_config migration completed successfully!'; END $$;
