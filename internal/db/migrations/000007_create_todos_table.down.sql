DELETE FROM status_enum WHERE name = 'pending';
DELETE FROM status_enum WHERE name = 'done';

DELETE FROM features WHERE name = 'todos';

DROP TABLE IF EXISTS status_enum;
DROP TABLE IF EXISTS todos;
