-- name: GetFeatures :many
SELECT * FROM features;

-- name: GetFeaturesByMessageId :many
SELECT feature_name, count FROM messages_features WHERE message_id = ?;

-- name: CreateMessageFeature :exec
INSERT INTO messages_features (message_id, feature_name) VALUES (?, ?);

-- name: IncrementMessageFeatureCount :exec
UPDATE messages_features SET count = count + 1 WHERE message_id = ? AND feature_name = ?;

-- name: DecrementMessageFeatureCount :exec
UPDATE messages_features SET count = count - 1 WHERE message_id = ? AND feature_name = ?;

-- name: FeatureExists :one
SELECT EXISTS(
    SELECT 1 FROM features
    WHERE name = ?
) AS "exists";

-- name: MessageHasFeature :one
SELECT EXISTS(
    SELECT 1 FROM messages_features 
    WHERE message_id = ?
    AND feature_name = ?
) AS "exists";
