package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/djumpen/gridlogger/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const notificationDispatcherLockKey int64 = 34045102

type ProjectNotificationRepository struct {
	pool *pgxpool.Pool
}

func NewProjectNotificationRepository(pool *pgxpool.Pool) *ProjectNotificationRepository {
	return &ProjectNotificationRepository{pool: pool}
}

func (r *ProjectNotificationRepository) GetProjectNotificationSubscription(ctx context.Context, userID int64, projectID int) (bool, error) {
	const q = `
		SELECT is_active
		FROM project_notification_subscriptions
		WHERE user_id = $1 AND project_id = $2
	`

	var subscribed bool
	err := r.pool.QueryRow(ctx, q, userID, projectID).Scan(&subscribed)
	if err == nil {
		return subscribed, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func (r *ProjectNotificationRepository) SetProjectNotificationSubscription(ctx context.Context, userID int64, projectID int, subscribed bool) (bool, error) {
	const q = `
		INSERT INTO project_notification_subscriptions (user_id, project_id, is_active, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id, project_id) DO UPDATE
		SET is_active = EXCLUDED.is_active,
			updated_at = now()
		RETURNING is_active
	`

	var current bool
	err := r.pool.QueryRow(ctx, q, userID, projectID, subscribed).Scan(&current)
	if err != nil {
		return false, err
	}
	return current, nil
}

func (r *ProjectNotificationRepository) ListProjectIDsWithActiveSubscriptions(ctx context.Context) ([]int, error) {
	const q = `
		SELECT DISTINCT project_id
		FROM project_notification_subscriptions
		WHERE is_active = TRUE
		ORDER BY project_id ASC
	`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]int, 0)
	for rows.Next() {
		var projectID int
		if err := rows.Scan(&projectID); err != nil {
			return nil, err
		}
		out = append(out, projectID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *ProjectNotificationRepository) ListActiveSubscribersByProjectID(ctx context.Context, projectID int) ([]service.ProjectNotificationSubscriber, error) {
	const q = `
		SELECT s.user_id, t.telegram_id, t.username, t.first_name, t.last_name
		FROM project_notification_subscriptions s
		INNER JOIN users u ON u.id = s.user_id
		INNER JOIN telegram_accounts t ON t.telegram_id = u.telegram_id
		WHERE s.project_id = $1
		  AND s.is_active = TRUE
		  AND t.is_blocked = FALSE
		ORDER BY s.user_id ASC
	`

	rows, err := r.pool.Query(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]service.ProjectNotificationSubscriber, 0)
	for rows.Next() {
		var item service.ProjectNotificationSubscriber
		if err := rows.Scan(
			&item.UserID,
			&item.TelegramID,
			&item.Username,
			&item.FirstName,
			&item.LastName,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *ProjectNotificationRepository) GetProjectStatusState(ctx context.Context, projectID int) (service.ProjectStatusState, bool, error) {
	const q = `
		SELECT project_id, last_status, last_changed_at, last_notified_status, last_notified_at, updated_at
		FROM project_status_state
		WHERE project_id = $1
	`

	var state service.ProjectStatusState
	var lastNotifiedAt sql.NullTime
	err := r.pool.QueryRow(ctx, q, projectID).Scan(
		&state.ProjectID,
		&state.LastStatus,
		&state.LastChangedAt,
		&state.LastNotifiedStatus,
		&lastNotifiedAt,
		&state.UpdatedAt,
	)
	if err == nil {
		if lastNotifiedAt.Valid {
			ts := lastNotifiedAt.Time.UTC()
			state.LastNotifiedAt = &ts
		}
		return state, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return service.ProjectStatusState{}, false, nil
	}
	return service.ProjectStatusState{}, false, err
}

func (r *ProjectNotificationRepository) UpsertProjectStatusState(ctx context.Context, in service.ProjectStatusStateUpsert) error {
	const q = `
		INSERT INTO project_status_state (
			project_id,
			last_status,
			last_changed_at,
			last_notified_status,
			last_notified_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (project_id) DO UPDATE
		SET last_status = EXCLUDED.last_status,
			last_changed_at = EXCLUDED.last_changed_at,
			last_notified_status = EXCLUDED.last_notified_status,
			last_notified_at = EXCLUDED.last_notified_at,
			updated_at = now()
	`

	_, err := r.pool.Exec(
		ctx,
		q,
		in.ProjectID,
		in.LastStatus,
		in.LastChangedAt,
		in.LastNotifiedStatus,
		in.LastNotifiedAt,
	)
	return err
}

func (r *ProjectNotificationRepository) TryAcquireNotificationDispatcherLock(ctx context.Context) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, notificationDispatcherLockKey).Scan(&ok)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (r *ProjectNotificationRepository) ReleaseNotificationDispatcherLock(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `SELECT pg_advisory_unlock($1)`, notificationDispatcherLockKey)
	return err
}
