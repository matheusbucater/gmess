-- name: GetTodos :many
SELECT * FROM todos;

-- name: GetTodosOrderByCreatedAtASC :many
SELECT * FROM todos ORDER BY created_at ASC;

-- name: GetTodosOrderByCreatedAtDESC :many
SELECT * FROM todos ORDER BY created_at DESC;

-- name: GetTodosOrderByUpdatedAtASC :many
SELECT * FROM todos ORDER BY updated_at ASC;

-- name: GetTodosOrderByUpdatedAtDESC :many
SELECT * FROM todos ORDER BY updated_at DESC;

-- name: GetTodosOrderByStatusASC :many
SELECT * FROM todos ORDER BY status ASC;

-- name: GetTodosOrderByStatusDESC :many
SELECT * FROM todos ORDER BY status DESC;

-- name: GetTodoById :one
SELECT * FROM todos WHERE id = ?;

-- name: GetTodoAndMessageByTodoId :one
SELECT 
    sqlc.embed(todos),
    sqlc.embed(messages)
FROM todos
INNER JOIN messages ON todos.message_id = messages.id
WHERE todos.id = ? 
GROUP BY todos.id;

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
