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

func (r *ProjectNotificationRepository) CountActiveProjectNotificationSubscriptions(ctx context.Context, projectID int) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM project_notification_subscriptions
		WHERE project_id = $1
		  AND is_active = TRUE
	`

	var count int
	if err := r.pool.QueryRow(ctx, q, projectID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ProjectNotificationRepository) ListActiveSubscribedProjectsByUserID(ctx context.Context, userID int64) ([]service.Project, error) {
	const q = `
		SELECT p.id, p.name, p.slug, p.user_id, p.city, p.description, p.created_at
		FROM project_notification_subscriptions s
		INNER JOIN projects p ON p.id = s.project_id
		WHERE s.user_id = $1
		  AND s.is_active = TRUE
		ORDER BY p.id ASC
	`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]service.Project, 0)
	for rows.Next() {
		var p service.Project
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Slug,
			&p.UserID,
			&p.City,
			&p.Description,
			&p.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
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

func (r *ProjectNotificationRepository) ListTelegramBotGroupsByProjectID(ctx context.Context, ownerID int64, projectID int) ([]service.ProjectTelegramBotGroup, error) {
	const q = `
		SELECT u.id, t.telegram_id, t.chat_title, t.chat_type, t.username, s.created_at
		FROM project_notification_subscriptions s
		INNER JOIN users u ON u.id = s.user_id
		INNER JOIN telegram_accounts t ON t.telegram_id = u.telegram_id
		WHERE s.project_id = $1
		  AND s.is_active = TRUE
		  AND u.is_virtual = TRUE
		  AND u.owner_id = $2
		ORDER BY t.chat_title ASC, u.id ASC
	`

	rows, err := r.pool.Query(ctx, q, projectID, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]service.ProjectTelegramBotGroup, 0)
	for rows.Next() {
		var item service.ProjectTelegramBotGroup
		if err := rows.Scan(
			&item.VirtualUserID,
			&item.TelegramID,
			&item.Title,
			&item.ChatType,
			&item.Username,
			&item.AddedAt,
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

func (r *ProjectNotificationRepository) UpsertTelegramBotGroupSubscription(ctx context.Context, in service.ProjectTelegramBotGroupUpsert) (service.ProjectTelegramBotGroup, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return service.ProjectTelegramBotGroup{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const upsertTelegramAccount = `
		INSERT INTO telegram_accounts (
			telegram_id,
			username,
			first_name,
			last_name,
			photo_url,
			chat_type,
			chat_title,
			last_auth_date,
			last_login_at,
			updated_at
		) VALUES ($1, $2, '', '', '', $3, $4, 0, now(), now())
		ON CONFLICT (telegram_id) DO UPDATE
		SET username = EXCLUDED.username,
			chat_type = EXCLUDED.chat_type,
			chat_title = EXCLUDED.chat_title,
			updated_at = now()
	`
	if _, err := tx.Exec(ctx, upsertTelegramAccount, in.TelegramID, in.Username, in.ChatType, in.Title); err != nil {
		return service.ProjectTelegramBotGroup{}, err
	}

	var userID int64
	var ownerID sql.NullInt64
	var isVirtual bool
	const existingUserQ = `
		SELECT id, owner_id, is_virtual
		FROM users
		WHERE telegram_id = $1
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, existingUserQ, in.TelegramID).Scan(&userID, &ownerID, &isVirtual)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		const insertUserQ = `
			INSERT INTO users (telegram_id, is_virtual, owner_id, updated_at)
			VALUES ($1, TRUE, $2, now())
			RETURNING id
		`
		if err := tx.QueryRow(ctx, insertUserQ, in.TelegramID, in.OwnerID).Scan(&userID); err != nil {
			return service.ProjectTelegramBotGroup{}, err
		}
	case err != nil:
		return service.ProjectTelegramBotGroup{}, err
	default:
		if !isVirtual || !ownerID.Valid || ownerID.Int64 != in.OwnerID {
			return service.ProjectTelegramBotGroup{}, service.ErrTelegramBotGroupConflict
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET updated_at = now() WHERE id = $1`, userID); err != nil {
			return service.ProjectTelegramBotGroup{}, err
		}
	}

	const upsertSubscriptionQ = `
		INSERT INTO project_notification_subscriptions (user_id, project_id, is_active, updated_at)
		VALUES ($1, $2, TRUE, now())
		ON CONFLICT (user_id, project_id) DO UPDATE
		SET is_active = TRUE,
			updated_at = now()
	`
	if _, err := tx.Exec(ctx, upsertSubscriptionQ, userID, in.ProjectID); err != nil {
		return service.ProjectTelegramBotGroup{}, err
	}

	item, err := selectTelegramBotGroupByUserID(ctx, tx, userID, in.ProjectID)
	if err != nil {
		return service.ProjectTelegramBotGroup{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return service.ProjectTelegramBotGroup{}, err
	}
	return item, nil
}

func (r *ProjectNotificationRepository) RemoveTelegramBotGroupSubscription(ctx context.Context, ownerID int64, projectID int, virtualUserID int64) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var telegramID int64
	const virtualUserQ = `
		SELECT telegram_id
		FROM users
		WHERE id = $1
		  AND is_virtual = TRUE
		  AND owner_id = $2
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, virtualUserQ, virtualUserID, ownerID).Scan(&telegramID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	cmd, err := tx.Exec(
		ctx,
		`DELETE FROM project_notification_subscriptions WHERE user_id = $1 AND project_id = $2`,
		virtualUserID,
		projectID,
	)
	if err != nil {
		return false, err
	}
	if cmd.RowsAffected() == 0 {
		return false, nil
	}

	var remaining int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM project_notification_subscriptions WHERE user_id = $1`, virtualUserID).Scan(&remaining); err != nil {
		return false, err
	}
	if remaining == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, virtualUserID); err != nil {
			return false, err
		}
		if _, err := tx.Exec(
			ctx,
			`DELETE FROM telegram_accounts WHERE telegram_id = $1 AND NOT EXISTS (SELECT 1 FROM users WHERE telegram_id = $1)`,
			telegramID,
		); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
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

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func selectTelegramBotGroupByUserID(ctx context.Context, q queryRower, userID int64, projectID int) (service.ProjectTelegramBotGroup, error) {
	const query = `
		SELECT u.id, t.telegram_id, t.chat_title, t.chat_type, t.username, s.created_at
		FROM users u
		INNER JOIN telegram_accounts t ON t.telegram_id = u.telegram_id
		INNER JOIN project_notification_subscriptions s ON s.user_id = u.id AND s.project_id = $2
		WHERE u.id = $1
	`

	var item service.ProjectTelegramBotGroup
	err := q.QueryRow(ctx, query, userID, projectID).Scan(
		&item.VirtualUserID,
		&item.TelegramID,
		&item.Title,
		&item.ChatType,
		&item.Username,
		&item.AddedAt,
	)
	if err != nil {
		return service.ProjectTelegramBotGroup{}, err
	}
	return item, nil
}
