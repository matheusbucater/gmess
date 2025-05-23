-- name: GetFeatures :many
SELECT * FROM features;

-- name: GetFeaturesByMessageId :many
SELECT feature_name FROM messages_features WHERE message_id = ?;
