-- +goose Up
CREATE TABLE IF NOT EXISTS messages (
    id BIGINT PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    sender_id BIGINT NOT NULL,
    sender_username TEXT,
    sender_first_name TEXT,
    sender_last_name TEXT,
    message_text TEXT,
    message_date TIMESTAMP NOT NULL,
    is_outgoing BOOLEAN NOT NULL DEFAULT FALSE,
    is_channel_post BOOLEAN NOT NULL DEFAULT FALSE,
    is_group_message BOOLEAN NOT NULL DEFAULT FALSE
);

-- +goose Down
DROP TABLE IF EXISTS messages;