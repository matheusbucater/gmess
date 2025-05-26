-- name: GetNotifications :many
SELECT * FROM notifications;

-- name: CreateNotification :one
INSERT INTO notifications (message_id, type) VALUES (?, ?) RETURNING *;
