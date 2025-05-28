-- name: GetNotifications :many
SELECT * FROM notifications;

-- name: GetNotificationsOrderByCreatedAtASC :many
SELECT * FROM notifications ORDER BY created_at ASC;

-- name: GetNotificationsOrderByCreatedAtDESC :many
SELECT * FROM notifications ORDER BY created_at DESC;

-- name: GetNotificationsOrderByUpdatedAtASC :many
SELECT * FROM notifications ORDER BY updated_at ASC;

-- name: GetNotificationsOrderByUpdatedAtDESC :many
SELECT * FROM notifications ORDER BY updated_at DESC;

-- name: GetNotificationsOrderByTypeASC :many
SELECT * FROM notifications ORDER BY type ASC;

-- name: GetNotificationsOrderByTypeDESC :many
SELECT * FROM notifications ORDER BY type DESC;

-- name: GetNotificationAndMessageById :one
SELECT 
    sqlc.embed(notifications),
    sqlc.embed(messages)
FROM notifications
INNER JOIN messages ON messages.id = notifications.message_id
WHERE notifications.id = ? 
GROUP BY notifications.id;

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
