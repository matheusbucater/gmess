-- name: GetMessageById :one
SELECT * FROM messages WHERE id = ? LIMIT 1;

-- name: GetMessages :many
SELECT * FROM messages;

-- name: GetMessagesOrderByCreatedAtASC :many
SELECT * FROM messages ORDER BY created_at ASC;

-- name: GetMessagesOrderByCreatedAtDESC :many
SELECT * FROM messages ORDER BY created_at DESC;

-- name: GetMessagesOrderByUpdatedAtASC :many
SELECT * FROM messages ORDER BY updated_at ASC;

-- name: GetMessagesOrderByUpdatedAtDESC :many
SELECT * FROM messages ORDER BY updated_at DESC;

-- name: GetMessagesOrderByTextASC :many
SELECT * FROM messages ORDER BY text ASC;

-- name: GetMessagesOrderByTextDESC :many
SELECT * FROM messages ORDER BY text DESC;

-- name: CreateMessage :one
INSERT INTO messages (text) VALUES (?) RETURNING *;

-- name: MessageExists :one
SELECT EXISTS(
    SELECT 1 FROM messages 
    WHERE id = ?
) AS "exists";

-- name: UpdateMessage :one
UPDATE messages SET text = ?, updated_at = CURRENT_TIMESTAMP  WHERE id = ? RETURNING *;

-- name: DeleteMessage :exec
DELETE FROM messages WHERE id = ?;

-- name: CountMessages :one
SELECT count(*) FROM messages;
