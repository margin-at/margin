package postgres

import (
	"context"
	"time"

	"margin.at/internal/domain"
)

type NotificationRepository struct {
	db DB
}

func NewNotificationRepository(db DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) CreateNotification(ctx context.Context, n *domain.Notification) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO notifications (recipient_did, actor_did, type, subject_uri, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, n.RecipientDID, n.ActorDID, n.Type, n.SubjectURI, n.CreatedAt)
	return err
}

func (r *NotificationRepository) GetNotifications(ctx context.Context, recipientDID string, limit, offset int) ([]domain.Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, recipient_did, actor_did, type, subject_uri, created_at, read_at
		FROM notifications
		WHERE recipient_did = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, recipientDID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.RecipientDID, &n.ActorDID, &n.Type, &n.SubjectURI, &n.CreatedAt, &n.ReadAt); err != nil {
			continue
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *NotificationRepository) GetUnreadNotificationCount(ctx context.Context, recipientDID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE recipient_did = $1 AND read_at IS NULL
	`, recipientDID).Scan(&count)
	return count, err
}

func (r *NotificationRepository) MarkNotificationsRead(ctx context.Context, recipientDID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = $1 WHERE recipient_did = $2 AND read_at IS NULL
	`, time.Now(), recipientDID)
	return err
}
