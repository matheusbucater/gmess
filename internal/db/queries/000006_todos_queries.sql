-- name: GetTodos :many
SELECT * FROM todos;

-- name: GetTodoById :one
SELECT * FROM todos WHERE id = ?;

-- name: GetTodoByMessageId :one
SELECT * FROM todos WHERE message_id = ?;

-- name: TodoExists :one
SELECT EXISTS(
    SELECT 1 FROM todos 
    WHERE id = ?
) AS "exists";

-- name: CreateTodo :one
INSERT INTO todos (message_id) VALUES (?) RETURNING *;

-- name: UpdateTodoByMessageId :one
UPDATE todos SET status = ? WHERE message_id = ? RETURNING *;

-- name: UpdateTodoById :one
UPDATE todos SET status = ? WHERE id = ? RETURNING *;

-- name: DeleteTodoById :exec
DELETE FROM todos WHERE id = ?;

-- name: DeleteTodoByIdReturningMsgId :one
DELETE FROM todos WHERE id = ? RETURNING message_id;

-- name: DeleteTodoByMessageId :exec
DELETE FROM todos WHERE message_id = ?;
