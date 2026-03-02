CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS secret TEXT;

UPDATE projects
SET secret = encode(gen_random_bytes(32), 'hex')
WHERE secret IS NULL OR secret = '';

ALTER TABLE projects
    ALTER COLUMN secret SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_slug ON projects(slug);
