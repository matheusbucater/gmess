-- name: GetFeatures :many
SELECT * FROM features;

-- name: GetFeaturesByMessageId :many
SELECT feature_name, count FROM messages_features WHERE message_id = ?;

-- name: GetPrettyFeaturesByMessageId :one
SELECT GROUP_CONCAT(SUBSTR(feature_name, 1, 3), ', ')
FROM messages_features 
WHERE message_id = ? AND count > 0;

-- name: GetMessageAndFeatures :one
SELECT 
  messages.*,
  GROUP_CONCAT(messages_features.feature_name, ', ') AS features
FROM messages
INNER JOIN messages_features ON messages_features.message_id = messages.id
WHERE messages.id = ? AND messages_features.count > 0
GROUP BY messages.id;

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
