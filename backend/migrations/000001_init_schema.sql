-- +goose Up
-- +goose StatementBegin

-- Создание перечислений (enums)
CREATE TYPE user_role AS ENUM ('student', 'teacher', 'admin');
CREATE TYPE notification_type AS ENUM ('schedule_change', 'system', 'important');
CREATE TYPE schedule_source_type AS ENUM ('main', 'change');
CREATE TYPE schedule_change_type AS ENUM ('replacement', 'cancellation', 'addition');

-- Таблица пользователей
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role user_role NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE
);

-- Индекс для поиска по email
CREATE INDEX idx_users_email ON users(email);

-- Таблица студентов
CREATE TABLE students (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    group_name VARCHAR(50) NOT NULL,
    faculty VARCHAR(100),
    course INTEGER CHECK (course >= 1 AND course <= 4),
    student_number VARCHAR(50) UNIQUE
);

-- Индекс для поиска по группе
CREATE INDEX idx_students_group ON students(group_name);

-- Таблица преподавателей
CREATE TABLE teachers (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    department VARCHAR(100),
    position VARCHAR(100),
    teacher_id VARCHAR(50) UNIQUE -- Внутренний ID преподавателя в колледже
);

-- Индекс для поиска по ФИО
CREATE INDEX idx_teachers_full_name ON teachers(full_name);

-- Таблица снапшотов расписания
CREATE TABLE schedule_snapshots (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    data JSONB NOT NULL, -- Используем JSONB для эффективного поиска и индексации
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    source_url TEXT,
    is_active BOOLEAN DEFAULT FALSE
);

-- Индексы для поиска по периодам
CREATE INDEX idx_schedule_snapshots_period ON schedule_snapshots(period_start, period_end);
CREATE INDEX idx_schedule_snapshots_active ON schedule_snapshots(is_active);

-- Таблица изменений в расписании
CREATE TABLE schedule_changes (
    id UUID PRIMARY KEY,
    snapshot_id UUID REFERENCES schedule_snapshots(id) ON DELETE SET NULL, -- Может быть NULL, если создано вручную
    group_name VARCHAR(50) NOT NULL,
    date DATE NOT NULL,
    time_start TIME WITHOUT TIME ZONE NOT NULL,
    time_end TIME WITHOUT TIME ZONE NOT NULL,
    subject VARCHAR(255) NOT NULL,
    teacher VARCHAR(255),
    classroom VARCHAR(50),
    change_type schedule_change_type NOT NULL,
    original_subject VARCHAR(255), -- Для типа 'replacement'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);

-- Индексы для поиска изменений
CREATE INDEX idx_schedule_changes_date_group ON schedule_changes(date, group_name);
CREATE INDEX idx_schedule_changes_active ON schedule_changes(is_active);

-- Таблица текущего расписания (актуальное на данный момент)
CREATE TABLE current_schedule (
    id UUID PRIMARY KEY,
    group_name VARCHAR(50) NOT NULL,
    date DATE NOT NULL,
    time_start TIME WITHOUT TIME ZONE NOT NULL,
    time_end TIME WITHOUT TIME ZONE NOT NULL,
    subject VARCHAR(255) NOT NULL,
    teacher VARCHAR(255),
    classroom VARCHAR(50),
    source_type schedule_source_type NOT NULL,
    source_id UUID NOT NULL, -- ID снапшота или изменения
    is_active BOOLEAN DEFAULT TRUE
);

-- Индексы для быстрого поиска текущего расписания
CREATE INDEX idx_current_schedule_date_group ON current_schedule(date, group_name);
CREATE INDEX idx_current_schedule_active ON current_schedule(is_active);

-- Таблица уведомлений
CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    type notification_type NOT NULL,
    related_group VARCHAR(50),
    related_date DATE,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для уведомлений
CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;

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
