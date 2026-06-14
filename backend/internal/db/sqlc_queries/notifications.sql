-- name: CreateNotification :exec
INSERT INTO notifications (recipient_did, actor_did, type, subject_uri, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetNotifications :many
SELECT id, recipient_did, actor_did, type, subject_uri, created_at, read_at
FROM notifications
WHERE recipient_did = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetUnreadNotificationCount :one
SELECT COUNT(*) FROM notifications WHERE recipient_did = $1 AND read_at IS NULL;

-- name: MarkNotificationsRead :exec
UPDATE notifications SET read_at = $1 WHERE recipient_did = $2 AND read_at IS NULL;
