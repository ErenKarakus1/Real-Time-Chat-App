ALTER TABLE conversation_participants
ADD COLUMN last_read_at TIMESTAMPTZ;

CREATE INDEX idx_conversation_participants_last_read_at
    ON conversation_participants(conversation_id, user_id, last_read_at);
