-- Migration: 000008_user_roles
-- Description: Remove role column from users table

DO $$ BEGIN RAISE NOTICE '[Migration 000008] Removing role column from users table...'; END $$;

ALTER TABLE users DROP COLUMN IF EXISTS role;

DO $$ BEGIN RAISE NOTICE '[Migration 000008] User roles rollback completed successfully!'; END $$;
