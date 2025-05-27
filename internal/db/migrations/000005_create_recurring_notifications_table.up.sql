CREATE TABLE week_day_enum (
    week_day TEXT PRIMARY KEY NOT NULL,
    seq INTEGER
);

CREATE TABLE recurring_notifications (
    notification_id INTEGER PRIMARY KEY REFERENCES notifications(id) ON DELETE CASCADE,
    trigger_at_time TEXT CHECK(trigger_at_time GLOB '??-??-??')
);

CREATE TABLE recurring_notification_days (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recurring_notification_id INTEGER NOT NULL REFERENCES recurring_notifications(notification_id) ON DELETE CASCADE,
    week_day TEXT NOT NULL DEFAULT ('monday') REFERENCES week_day_enum(week_day)
);

INSERT INTO type_enum (type, seq) VALUES ('recurring', 2);

INSERT INTO week_day_enum (week_day, seq) VALUES ('sunday', 1);
INSERT INTO week_day_enum (week_day, seq) VALUES ('monday', 2);
INSERT INTO week_day_enum (week_day, seq) VALUES ('tuesday', 3);
INSERT INTO week_day_enum (week_day, seq) VALUES ('wednesday', 4);
INSERT INTO week_day_enum (week_day, seq) VALUES ('thursday', 5);
INSERT INTO week_day_enum (week_day, seq) VALUES ('friday', 6);
INSERT INTO week_day_enum (week_day, seq) VALUES ('saturday', 7);
