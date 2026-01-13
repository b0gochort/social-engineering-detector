-- Remove indexes
DROP INDEX IF EXISTS idx_users_parent_id;
DROP INDEX IF EXISTS idx_users_telegram_id;

-- Remove columns
ALTER TABLE users DROP COLUMN IF EXISTS telegram_id;
ALTER TABLE users DROP COLUMN IF EXISTS parent_id;
