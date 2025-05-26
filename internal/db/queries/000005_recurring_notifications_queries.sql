-- name: GetSimpleNotificationByNotificationId :one
SELECT * FROM simple_notifications WHERE notification_id = ?;

-- name: CreateSimpleNotification :exec
INSERT INTO simple_notifications (notification_id, trigger_at) VALUES (?, ?);
