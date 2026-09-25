ALTER TABLE conversation_participants
    ADD COLUMN role TEXT NOT NULL DEFAULT 'member';

UPDATE conversation_participants cp
SET role = 'owner'
FROM conversations c
WHERE cp.conversation_id = c.id
    AND cp.user_id = c.created_by;

ALTER TABLE conversation_participants
    ADD CONSTRAINT conversation_participants_role_check
    CHECK (role IN ('owner', 'admin', 'member'));
