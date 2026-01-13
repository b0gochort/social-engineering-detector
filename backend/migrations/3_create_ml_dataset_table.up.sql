-- Create ML dataset table for storing all messages (both neutral and threats)
-- This table is NOT encrypted to allow easy access for ML model training
CREATE TABLE ml_dataset (
    id SERIAL PRIMARY KEY,

    -- Message content (plain text, NOT encrypted)
    message_text TEXT NOT NULL,

    -- Annotation from LLM
    category_id INTEGER NOT NULL,
    category_name TEXT NOT NULL,
    justification TEXT,

    -- Model metadata
    provider TEXT NOT NULL,           -- groq, gemini, etc.
    model_version TEXT NOT NULL,      -- llama-3.3-70b-versatile, gemini-2.0-flash-exp
    annotated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Optional: reference to original message (if available)
    original_message_id INTEGER REFERENCES messages(id) ON DELETE SET NULL,

    -- Flag for validation
    is_validated BOOLEAN DEFAULT FALSE,
    validated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    validated_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    source TEXT DEFAULT 'telegram',   -- telegram, manual, synthetic
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for faster queries
CREATE INDEX idx_ml_dataset_category_id ON ml_dataset(category_id);
CREATE INDEX idx_ml_dataset_provider ON ml_dataset(provider);
CREATE INDEX idx_ml_dataset_is_validated ON ml_dataset(is_validated);
CREATE INDEX idx_ml_dataset_annotated_at ON ml_dataset(annotated_at);

-- Comment explaining the purpose
COMMENT ON TABLE ml_dataset IS 'Dataset for ML model training. Contains ALL messages (neutral + threats) in plain text for training purposes. NOT encrypted.';
