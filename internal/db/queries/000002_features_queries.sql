-- name: GetFeatures :many
SELECT * FROM features;

-- name: GetFeaturesByMessageId :many
SELECT feature_name FROM messages_features WHERE message_id = ?;

-- name: CreateMessageFeature :exec
INSERT INTO messages_features (message_id, feature_name) VALUES (?, ?);
