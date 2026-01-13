-- Add source column to chats table to distinguish between Telegram and VK
ALTER TABLE chats ADD COLUMN source VARCHAR(50) DEFAULT 'telegram' NOT NULL;

-- Add VK-specific identifier for chats (peer_id for VK conversations)
ALTER TABLE chats ADD COLUMN vk_peer_id BIGINT;

-- Add type column to chats to store conversation type (user, chat, channel, group)
ALTER TABLE chats ADD COLUMN chat_type VARCHAR(50);

-- Make telegram_id nullable since VK chats won't have it
ALTER TABLE chats ALTER COLUMN telegram_id DROP NOT NULL;

-- Add unique constraint for VK chats
CREATE UNIQUE INDEX unique_vk_chat ON chats(vk_peer_id) WHERE vk_peer_id IS NOT NULL;

-- Update existing chats to have chat_type
UPDATE chats SET chat_type = CASE WHEN is_group THEN 'group' ELSE 'user' END WHERE chat_type IS NULL;

-- Add source column to messages table
ALTER TABLE messages ADD COLUMN source VARCHAR(50) DEFAULT 'telegram' NOT NULL;

-- Add message type column (message, post, comment)
ALTER TABLE messages ADD COLUMN message_type VARCHAR(50) DEFAULT 'message' NOT NULL;

-- Add VK-specific message identifier
ALTER TABLE messages ADD COLUMN vk_message_id BIGINT;

-- Make telegram_message_id nullable since VK messages won't have it
ALTER TABLE messages ALTER COLUMN telegram_message_id DROP NOT NULL;

-- Add unique constraint for VK messages
CREATE UNIQUE INDEX unique_vk_message ON messages(chat_id, vk_message_id) WHERE vk_message_id IS NOT NULL;
