-- Remove unique constraint for VK messages
DROP INDEX IF EXISTS unique_vk_message;

-- Make telegram_message_id NOT NULL again
ALTER TABLE messages ALTER COLUMN telegram_message_id SET NOT NULL;

-- Remove VK-specific message identifier
ALTER TABLE messages DROP COLUMN IF EXISTS vk_message_id;

-- Remove message type column
ALTER TABLE messages DROP COLUMN IF EXISTS message_type;

-- Remove source column from messages
ALTER TABLE messages DROP COLUMN IF EXISTS source;

-- Remove unique constraint for VK chats
DROP INDEX IF EXISTS unique_vk_chat;

-- Make telegram_id NOT NULL again
ALTER TABLE chats ALTER COLUMN telegram_id SET NOT NULL;

-- Remove chat type column
ALTER TABLE chats DROP COLUMN IF EXISTS chat_type;

-- Remove VK-specific identifier
ALTER TABLE chats DROP COLUMN IF EXISTS vk_peer_id;

-- Remove source column from chats
ALTER TABLE chats DROP COLUMN IF EXISTS source;
