package db

import (
	"context"
	"time"

	"margin.at/internal/db/sqlcdb"
)

func (db *DB) CreateNotification(ctx context.Context, n *Notification) error {
	return db.q.CreateNotification(ctx, sqlcdb.CreateNotificationParams{
		RecipientDid: n.RecipientDID,
		ActorDid:     n.ActorDID,
		Type:         n.Type,
		SubjectUri:   n.SubjectURI,
		CreatedAt:    n.CreatedAt,
	})
}

func (db *DB) GetNotifications(ctx context.Context, recipientDID string, limit, offset int) ([]Notification, error) {
	rows, err := db.q.GetNotifications(ctx, sqlcdb.GetNotificationsParams{
		RecipientDid: recipientDID,
		Limit:        int32(limit),
		Offset:       int32(offset),
	})
	if err != nil {
		return nil, err
	}

	var notifications []Notification
	for _, r := range rows {
		notifications = append(notifications, Notification{
			ID:           int(r.ID),
			RecipientDID: r.RecipientDid,
			ActorDID:     r.ActorDid,
			Type:         r.Type,
			SubjectURI:   r.SubjectUri,
			CreatedAt:    r.CreatedAt,
			ReadAt:       r.ReadAt,
		})
	}
	return notifications, nil
}

func (db *DB) GetUnreadNotificationCount(ctx context.Context, recipientDID string) (int, error) {
	count, err := db.q.GetUnreadNotificationCount(ctx, recipientDID)
	return int(count), err
}

func (db *DB) MarkNotificationsRead(ctx context.Context, recipientDID string) error {
	now := time.Now()
	return db.q.MarkNotificationsRead(ctx, sqlcdb.MarkNotificationsReadParams{
		ReadAt:       &now,
		RecipientDid: recipientDID,
	})
}
