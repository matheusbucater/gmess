-- name: GetRecurringNotificationByNotificationId :one
SELECT * FROM recurring_notifications WHERE notification_id = ?;

-- name: GetRecurringNotificationDaysByNotificationId :many
SELECT * FROM recurring_notification_days WHERE recurring_notification_id = ?;

-- name: CreateRecurringNotification :one
INSERT INTO recurring_notifications (notification_id, trigger_at_time) VALUES (?, ?) RETURNING *;

-- name: CreateRecurringNotificationDay :exec
INSERT INTO recurring_notification_days (recurring_notification_id, week_day) VALUES (?, ?);

-- name: RecurringNotificationHasDay :one
SELECT EXISTS(
    SELECT 1 FROM recurring_notification_days 
    WHERE recurring_notification_id = ?
    AND week_day = ?
) AS "exists";
