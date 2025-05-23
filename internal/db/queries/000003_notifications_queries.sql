-- name: GetNotifications :many
SELECT * FROM notifications;

-- name: GetSimpleNotificationByNotificationId :one
SELECT * FROM simple_notifications WHERE notification_id = ?;

-- name: CreateNotification :one
INSERT INTO notifications (message_id, type) VALUES (?, ?) RETURNING *;

-- name: CreateSimpleNotification :exec
INSERT INTO simple_notifications (notification_id, trigger_at) VALUES (?, ?);
