CREATE TABLE IF NOT EXISTS project_notification_subscriptions (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, project_id)
);

CREATE INDEX IF NOT EXISTS idx_project_notification_subscriptions_project_active
    ON project_notification_subscriptions(project_id, is_active);

CREATE TABLE IF NOT EXISTS project_status_state (
    project_id BIGINT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    last_status TEXT NOT NULL DEFAULT 'unknown',
    last_changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_notified_status TEXT NOT NULL DEFAULT '',
    last_notified_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_project_status_state_last_changed_at
    ON project_status_state(last_changed_at DESC);
