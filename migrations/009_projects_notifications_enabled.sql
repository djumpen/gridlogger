ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS notifications_enabled BOOLEAN;

UPDATE projects
SET notifications_enabled = TRUE
WHERE notifications_enabled IS NULL;

ALTER TABLE projects
    ALTER COLUMN notifications_enabled SET DEFAULT TRUE,
    ALTER COLUMN notifications_enabled SET NOT NULL;
