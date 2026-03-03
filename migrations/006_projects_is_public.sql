ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS is_public BOOLEAN;

UPDATE projects
SET is_public = TRUE
WHERE is_public IS NULL;

ALTER TABLE projects
    ALTER COLUMN is_public SET DEFAULT TRUE,
    ALTER COLUMN is_public SET NOT NULL;
