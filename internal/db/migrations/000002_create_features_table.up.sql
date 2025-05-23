CREATE TABLE features (
    name TEXT PRIMARY KEY NOT NULL,
    seq INTEGER NOT NULL
);

CREATE TABLE messages_features (
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    feature_name TEXT NOT NULL REFERENCES features(name),
    PRIMARY KEY (message_id, feature_name)
);

-- Every time a new feature is created, the feature need to be added to the features table
-- ex.: INSERT INTO features (name, seq) VALUES ('notifications', 1);
--      INSERT INTO features (name, seq) VALUES ('todos', 2);
--      INSERT INTO features (name, seq) VALUES ('groups', 3);
--      INSERT INTO features (name, seq) VALUES ('tags', 4;);

-- When a feature is deleted, the feature need to be removed from the features table
-- ex.: DELETE FROM features WHERE name = 'notifications';
--      DELETE FROM features WHERE name = 'todos';
--      DELETE FROM features WHERE name = 'groups';
--      DELETE FROM features WHERE name = 'tags';
