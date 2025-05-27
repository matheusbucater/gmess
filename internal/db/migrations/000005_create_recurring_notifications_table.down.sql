DELETE FROM type_enum WHERE type = 'recurring';

DROP TABLE IF EXISTS recurring_notification_days;
DROP TABLE IF EXISTS recurring_notifications;
DROP TABLE IF EXISTS week_day_enum;
