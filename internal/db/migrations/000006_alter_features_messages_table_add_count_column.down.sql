PRAGMA foreign_keys=off;

BEGIN TRANSACTION;

ALTER TABLE messages_features RENAME TO _messages_features_old;

CREATE TABLE messages_features (
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    feature_name TEXT NOT NULL REFERENCES features(name) ON DELETE CASCADE,
    PRIMARY KEY (message_id, feature_name)
);

INSERT INTO messages_features (message_id, feature_name)
  SELECT message_id, feature_name
  FROM _messages_features_old;

COMMIT;

PRAGMA foreign_keys=on;
