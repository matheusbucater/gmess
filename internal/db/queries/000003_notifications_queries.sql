-- name: GetNotifications :many
SELECT * FROM notifications;

-- name: CreateNotification :one
INSERT INTO notifications (message_id, type) VALUES (?, ?) RETURNING *;

-- name: CreateSingleNotification :exec
INSERT INTO single_notifications (notification_id, trigger_at) VALUES (?, ?);

