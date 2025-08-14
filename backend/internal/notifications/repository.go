// Package notifications предоставляет доступ к хранению уведомлений
package notifications

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository предоставляет доступ к хранению уведомлений
type Repository struct {
	db *sql.DB
}

// NewRepository создает новый репозиторий уведомлений
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateNotification создает новое уведомление
func (r *Repository) CreateNotification(ctx context.Context, notification *Notification) error {
	query := `
		INSERT INTO notifications 
		(id, user_id, title, message, type, related_group, related_date, is_read)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at`

	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		notification.ID,
		notification.UserID,
		notification.Title,
		notification.Message,
		notification.Type,
		notification.RelatedGroup,
		notification.RelatedDate,
		notification.IsRead).
		Scan(&createdAt)

	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	notification.CreatedAt = createdAt
	return nil
}

// GetUnreadNotifications получает непрочитанные уведомления для пользователя
func (r *Repository) GetUnreadNotifications(ctx context.Context, userID uuid.UUID) ([]Notification, error) {
	query := `
		SELECT id, user_id, title, message, type, related_group, related_date, is_read, created_at
		FROM notifications
		WHERE user_id = $1 AND is_read = false
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread notifications: %w", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notification Notification
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Title,
			&notification.Message,
			&notification.Type,
			&notification.RelatedGroup,
			&notification.RelatedDate,
			&notification.IsRead,
			&notification.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, notification)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return notifications, nil
}

// MarkAsRead помечает уведомление как прочитанное
func (r *Repository) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	query := `UPDATE notifications SET is_read = true WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, notificationID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}
