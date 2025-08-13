-- +goose Up
-- +goose StatementBegin
SELECT 'down migration is not implemented for this version'; 
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Удаление таблиц в обратном порядке создания
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS current_schedule;
DROP TABLE IF EXISTS schedule_changes;
DROP TABLE IF EXISTS schedule_snapshots;
DROP TABLE IF EXISTS teachers;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS users;

-- Удаление перечислений
DROP TYPE IF EXISTS schedule_change_type;
DROP TYPE IF EXISTS schedule_source_type;
DROP TYPE IF EXISTS notification_type;
DROP TYPE IF EXISTS user_role;
-- +goose StatementEnd
