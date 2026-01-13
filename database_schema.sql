-- ============================================================================
-- Social Engineering Detector - Database Schema
-- ============================================================================
-- Полная схема базы данных с учетом всех миграций (1-7)
-- Версия: 1.0
-- Дата: 2025-12-16
-- ============================================================================

-- ============================================================================
-- TABLE: users
-- Описание: Пользователи системы (родители и дети)
-- ============================================================================
CREATE TABLE users (
    -- Primary Key
    id SERIAL PRIMARY KEY,

    -- Authentication
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,  -- 'admin', 'parent', 'child'

    -- Encryption
    dk_encrypted TEXT NOT NULL,  -- Data key для шифрования данных

    -- Parent-Child Relationship (Migration 4)
    parent_id INTEGER REFERENCES users(id),  -- NULL для parent, заполнено для child

    -- Telegram Integration (Migration 4)
    telegram_id BIGINT,  -- ID Telegram аккаунта пользователя

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_users_parent_id ON users(parent_id);


-- ============================================================================
-- TABLE: chats
-- Описание: Telegram чаты, которые мониторятся системой
-- ============================================================================
CREATE TABLE chats (
    -- Primary Key
    id SERIAL PRIMARY KEY,

    -- Telegram Info
    telegram_id BIGINT NOT NULL UNIQUE,  -- ID чата в Telegram
    name TEXT NOT NULL,                   -- Название чата
    is_group BOOLEAN NOT NULL,            -- TRUE для групп, FALSE для личных

    -- Monitoring
    monitoring_active BOOLEAN NOT NULL,   -- Включен ли мониторинг

    -- Collector State (Migration 2)
    last_collected_message_id BIGINT DEFAULT 0  -- ID последнего собранного сообщения
);


-- ============================================================================
-- TABLE: messages
-- Описание: Все собранные сообщения из Telegram (зашифрованные)
-- ============================================================================
CREATE TABLE messages (
    -- Primary Key
    id SERIAL PRIMARY KEY,

    -- Relations
    chat_id INTEGER NOT NULL REFERENCES chats(id),

    -- Telegram Info
    telegram_message_id BIGINT NOT NULL,  -- ID сообщения в Telegram
    sender_username TEXT NOT NULL,        -- Отправитель
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Content (ENCRYPTED)
    content_encrypted TEXT NOT NULL  -- Зашифрованный текст сообщения
);


-- ============================================================================
-- TABLE: incidents
-- Описание: Обнаруженные угрозы социальной инженерии
-- ============================================================================
CREATE TABLE incidents (
    -- Primary Key
    id SERIAL PRIMARY KEY,

    -- Relations
    message_id INTEGER NOT NULL REFERENCES messages(id),

    -- ML Classification
    threat_type TEXT NOT NULL,          -- Primary threat category (from v2 model)
    model_confidence REAL NOT NULL,     -- Уверенность модели (0.0-1.0)

    -- Dual Model Categories (Migration 7)
    v2_category_id INTEGER,             -- Category ID from v2 model (1-9, accuracy 67.96%)
    v4_category_id INTEGER,             -- Category ID from v4 model (1-9, accuracy 64.00%)
    models_agree BOOLEAN,               -- Whether both models predicted the same category

    -- Status
    status TEXT NOT NULL,  -- 'new', 'reviewed', 'resolved', 'false_positive'

    -- Access Control (Migration 6)
    access_granted BOOLEAN NOT NULL DEFAULT FALSE,  -- Разрешен ли доступ родителю
    current_access_request_id INTEGER REFERENCES access_requests(id) ON DELETE SET NULL,

    -- Content
    summary_encrypted TEXT NOT NULL,  -- Краткое описание угрозы (зашифровано)

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_incidents_access_granted ON incidents(access_granted);

-- Comments
COMMENT ON COLUMN incidents.threat_type IS 'Primary threat category (from v2 model)';
COMMENT ON COLUMN incidents.v2_category_id IS 'Category ID from v2 model (1-9, accuracy 67.96%)';
COMMENT ON COLUMN incidents.v4_category_id IS 'Category ID from v4 model (1-9, accuracy 64.00%)';
COMMENT ON COLUMN incidents.models_agree IS 'Whether both models predicted the same category';


-- ============================================================================
-- TABLE: access_requests
-- Описание: Запросы родителей на доступ к инцидентам детей
-- ============================================================================
CREATE TABLE access_requests (
    -- Primary Key
    id SERIAL PRIMARY KEY,

    -- Relations
    incident_id INTEGER NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    parent_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    child_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- 'pending', 'approved', 'rejected'

    -- Timestamps
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    responded_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_access_requests_incident_id ON access_requests(incident_id);
CREATE INDEX idx_access_requests_parent_id ON access_requests(parent_id);
CREATE INDEX idx_access_requests_child_id ON access_requests(child_id);
CREATE INDEX idx_access_requests_status ON access_requests(status);

-- Constraints
ALTER TABLE access_requests ADD CONSTRAINT check_status
    CHECK (status IN ('pending', 'approved', 'rejected'));


-- ============================================================================
-- TABLE: ml_dataset
-- Описание: Dataset для обучения ML модели (НЕ зашифрован)
-- ============================================================================
CREATE TABLE ml_dataset (
    -- Primary Key
    id SERIAL PRIMARY KEY,

    -- Message Content (PLAIN TEXT, NOT ENCRYPTED)
    message_text TEXT NOT NULL,

    -- Annotation from LLM
    category_id INTEGER NOT NULL,        -- 1-9 (см. категории ниже)
    category_name TEXT NOT NULL,
    justification TEXT,                  -- Объяснение от LLM

    -- Model Metadata
    provider TEXT NOT NULL,              -- 'groq', 'gemini', etc.
    model_version TEXT NOT NULL,         -- 'llama-3.3-70b-versatile', 'gemini-2.0-flash-exp'
    annotated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Optional: Reference to Original Message
    original_message_id INTEGER REFERENCES messages(id) ON DELETE SET NULL,

    -- Validation
    is_validated BOOLEAN DEFAULT FALSE,
    validated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    validated_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    source TEXT DEFAULT 'telegram',      -- 'telegram', 'manual', 'synthetic', 'real'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_ml_dataset_category_id ON ml_dataset(category_id);
CREATE INDEX idx_ml_dataset_provider ON ml_dataset(provider);
CREATE INDEX idx_ml_dataset_is_validated ON ml_dataset(is_validated);
CREATE INDEX idx_ml_dataset_annotated_at ON ml_dataset(annotated_at);

-- Comment
COMMENT ON TABLE ml_dataset IS 'Dataset for ML model training. Contains ALL messages (neutral + threats) in plain text for training purposes. NOT encrypted.';


-- ============================================================================
-- TABLE: audit_log
-- Описание: Лог действий пользователей для аудита
-- ============================================================================
CREATE TABLE audit_log (
    -- Primary Key
    id SERIAL PRIMARY KEY,

    -- Relations
    user_id INTEGER REFERENCES users(id),

    -- Action Details
    action_type TEXT NOT NULL,  -- 'login', 'view_incident', 'approve_access', etc.
    details TEXT NOT NULL,      -- JSON с деталями действия

    -- Metadata
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ip_address TEXT
);


-- ============================================================================
-- ML CATEGORIES (1-9)
-- ============================================================================
/*
1. Склонение к сексуальным действиям (Груминг)
2. Угрозы, шантаж, вымогательство
3. Физическое насилие/Буллинг
4. Склонение к суициду/Самоповреждению
5. Склонение к опасным играм/действиям
6. Пропаганда запрещенных веществ
7. Финансовое мошенничество
8. Сбор личных данных (Фишинг)
9. Нейтральное общение
*/


-- ============================================================================
-- RELATIONSHIPS SUMMARY
-- ============================================================================
/*
users (parent_id) -> users (id)           -- Parent-Child relationship
messages (chat_id) -> chats (id)          -- Message belongs to Chat
incidents (message_id) -> messages (id)   -- Incident detected in Message
incidents (current_access_request_id) -> access_requests (id)
access_requests (incident_id) -> incidents (id)
access_requests (parent_id) -> users (id)
access_requests (child_id) -> users (id)
ml_dataset (original_message_id) -> messages (id)  -- Optional link
ml_dataset (validated_by) -> users (id)
audit_log (user_id) -> users (id)
*/


-- ============================================================================
-- ENCRYPTION NOTES
-- ============================================================================
/*
ENCRYPTED FIELDS (using AES-GCM with per-user data keys):
- users.dk_encrypted         -- User's data encryption key (encrypted with master key)
- messages.content_encrypted -- Message content
- incidents.summary_encrypted -- Incident summary

NOT ENCRYPTED (for ML training):
- ml_dataset.message_text    -- Plain text needed for model training
*/


-- ============================================================================
-- INDEXES SUMMARY
-- ============================================================================
/*
users:
  - idx_users_telegram_id
  - idx_users_parent_id

incidents:
  - idx_incidents_access_granted

access_requests:
  - idx_access_requests_incident_id
  - idx_access_requests_parent_id
  - idx_access_requests_child_id
  - idx_access_requests_status

ml_dataset:
  - idx_ml_dataset_category_id
  - idx_ml_dataset_provider
  - idx_ml_dataset_is_validated
  - idx_ml_dataset_annotated_at
*/
