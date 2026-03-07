CREATE TABLE IF NOT EXISTS dtek_groups (
    project_id BIGINT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    region_id INTEGER NOT NULL,
    region_name TEXT NOT NULL DEFAULT '',
    dso_id INTEGER NOT NULL,
    dso_name TEXT NOT NULL DEFAULT '',
    street_id INTEGER NOT NULL,
    street_name TEXT NOT NULL DEFAULT '',
    house_id INTEGER NOT NULL,
    house_name TEXT NOT NULL DEFAULT '',
    group_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_dtek_groups_group_key ON dtek_groups(group_key);
