CREATE TABLE simple_notifications (
    notification_id INTEGER PRIMARY KEY REFERENCES notifications(id) ON DELETE CASCADE,
    trigger_at TIMESTAMP NOT NULL
);

INSERT INTO type_enum (type, seq) VALUES ('simple', 1);
