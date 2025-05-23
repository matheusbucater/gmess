-- name: GetFeatures :many
SELECT * FROM features;

-- name: GetFeaturesByMessageId :many
SELECT feature_name FROM messages_features WHERE message_id = ?;

-- name: CreateMessageFeature :exec
INSERT INTO messages_features (message_id, feature_name) VALUES (?, ?);

-- name: MessageHasFeature :one
SELECT EXISTS(
    SELECT 1 FROM messages_features 
    WHERE message_id = ?
    AND feature_name = ?
) AS "exists";
