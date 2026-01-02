-- Migration: 000008_user_roles
-- Description: Add role column to users table

DO $$ BEGIN RAISE NOTICE '[Migration 000008] Adding role column to users table...'; END $$;

ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'user';

-- Update existing super admins to have 'admin' role as well
UPDATE users SET role = 'admin' WHERE can_access_all_tenants = true;

COMMENT ON COLUMN users.role IS 'Role of the user within the tenant (admin, user)';

DO $$ BEGIN RAISE NOTICE '[Migration 000008] User roles migration completed successfully!'; END $$;
