CREATE TABLE type_enum (
    type TEXT PRIMARY KEY NOT NULL,
    seq INTEGER
);

CREATE TABLE week_day_enum (
    week_day TEXT PRIMARY KEY NOT NULL,
    seq INTEGER
);

CREATE TABLE notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    type TEXT NOT NULL DEFAULT ('single') REFERENCES type_enum(type),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE single_notifications (
    notification_id INTEGER PRIMARY KEY REFERENCES notifications(id) ON DELETE CASCADE,
    trigger_at TIMESTAMP NOT NULL
);

CREATE TABLE multi_notifications (
    notification_id INTEGER PRIMARY KEY REFERENCES notifications(id) ON DELETE CASCADE
);

CREATE TABLE multi_notification_dates (
    multi_notification_id INTEGER REFERENCES multi_notifications(id) ON DELETE CASCADE,
    trigger_at TIMESTAMP NOT NULL,
    PRIMARY KEY (multi_notification_id, trigger_at)
);

CREATE TABLE recurring_notifications (
    notification_id INTEGER PRIMARY KEY REFERENCES notifications(id) ON DELETE CASCADE,
    trigger_at_time TIME NOT NULL
);

CREATE TABLE recurring_notification_days (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recurring_notification_id INTEGER NOT NULL REFERENCES recurring_notifications(notification_id) ON DELETE CASCADE,
    week_day TEXT NOT NULL DEFAULT ('monday') REFERENCES week_day_enum(week_day)
);

INSERT INTO features (name, seq) VALUES ('notifications', 1);

INSERT INTO type_enum (type, seq) VALUES ('single', 1);
INSERT INTO type_enum (type, seq) VALUES ('multi', 2);
INSERT INTO type_enum (type, seq) VALUES ('recurring', 3);

INSERT INTO week_day_enum (week_day, seq) VALUES ('sunday', 1);
INSERT INTO week_day_enum (week_day, seq) VALUES ('monday', 2);
INSERT INTO week_day_enum (week_day, seq) VALUES ('tuesday', 3);
INSERT INTO week_day_enum (week_day, seq) VALUES ('wednesday', 4);
INSERT INTO week_day_enum (week_day, seq) VALUES ('thursday', 5);
INSERT INTO week_day_enum (week_day, seq) VALUES ('friday', 6);
INSERT INTO week_day_enum (week_day, seq) VALUES ('saturday', 7);
