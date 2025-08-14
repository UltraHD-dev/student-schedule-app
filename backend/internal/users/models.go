// Package users определяет модели данных для работы с пользователями
// В соответствии с ТЗ: "User Management Service - управление пользователями"
package users

import (
	"time"

	"github.com/google/uuid"
)

// Role представляет роль пользователя в системе
type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
	RoleAdmin   Role = "admin"
)

// User представляет пользователя в системе
type User struct {
	ID        uuid.UUID  `db:"id"`
	Email     string     `db:"email"`
	Password  string     `db:"password_hash"` // This will store the hash
	Role      Role       `db:"role"`
	CreatedAt time.Time  `db:"created_at"`
	LastLogin *time.Time `db:"last_login"` // Pointer to handle NULL values
	IsActive  bool       `db:"is_active"`
}

// Student представляет дополнительную информацию для студента
type Student struct {
	UserID        uuid.UUID `db:"user_id"`
	GroupName     string    `db:"group_name"`
	Faculty       string    `db:"faculty"`
	Course        int       `db:"course"`
	StudentNumber string    `db:"student_number"`
}

// Teacher представляет дополнительную информацию для преподавателя
type Teacher struct {
	UserID     uuid.UUID `db:"user_id"`
	FullName   string    `db:"full_name"`
	Department string    `db:"department"`
	Position   string    `db:"position"`
	TeacherID  string    `db:"teacher_id"`
}
