-- Migration: 000006_admin_and_menu_config
-- Description: Rollback admin and menu_config changes

DELETE FROM users WHERE username = 'admin' AND can_access_all_tenants = true;
ALTER TABLE tenants DROP COLUMN IF EXISTS menu_config;
