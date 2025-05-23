DELETE FROM features WHERE name = 'notifications';

DROP TABLE IF EXISTS recurring_notification_days;
DROP TABLE IF EXISTS recurring_notifications;
DROP TABLE IF EXISTS multi_notification_dates;
DROP TABLE IF EXISTS multi_notifications;
DROP TABLE IF EXISTS single_notifications;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS week_day_enum;
DROP TABLE IF EXISTS type_enum;
