-- Add parent_id to users table for parent-child relationship
ALTER TABLE users ADD COLUMN parent_id INTEGER REFERENCES users(id);

-- Add telegram_id for linking Telegram account to user
ALTER TABLE users ADD COLUMN telegram_id BIGINT;

-- Add index on telegram_id for faster lookups
CREATE INDEX idx_users_telegram_id ON users(telegram_id);

-- Add index on parent_id
CREATE INDEX idx_users_parent_id ON users(parent_id);
