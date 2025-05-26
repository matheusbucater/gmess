-- name: GetNotifications :many
SELECT * FROM notifications;

-- name: CreateNotification :one
INSERT INTO notifications (message_id, type) VALUES (?, ?) RETURNING *;

-- name: GetNotificationById :one
SELECT * FROM notifications WHERE id = ?;

-- name: NotificationExists :one
SELECT EXISTS(
    SELECT 1 
    FROM notifications
    WHERE id = ?
) AS "exists";

-- name: DeleteNotificationById :exec
DELETE FROM notifications WHERE id = ?;

-- name: DeleteNotificationByIdReturningMsgId :one
DELETE FROM notifications WHERE id = ? RETURNING message_id;
