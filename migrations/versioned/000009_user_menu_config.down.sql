-- Migration: 000009_user_menu_config
-- Description: Remove menu_config column from users table

DO $$ BEGIN RAISE NOTICE '[Migration 000009] Removing menu_config column from users table...'; END $$;

ALTER TABLE users DROP COLUMN IF EXISTS menu_config;

DO $$ BEGIN RAISE NOTICE '[Migration 000009] User menu_config rollback completed successfully!'; END $$;
