package users

import (
	"time"

	"github.com/google/uuid"
)

// Role represents user role in the system
type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
	RoleAdmin   Role = "admin"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID  `db:"id"`
	Email     string     `db:"email"`
	Password  string     `db:"password_hash"` // This will store the hash
	Role      Role       `db:"role"`
	CreatedAt time.Time  `db:"created_at"`
	LastLogin *time.Time `db:"last_login"` // Pointer to handle NULL values
	IsActive  bool       `db:"is_active"`
}

// Student represents additional information for student users
type Student struct {
	UserID        uuid.UUID `db:"user_id"`
	GroupName     string    `db:"group_name"`
	Faculty       string    `db:"faculty"`
	Course        int       `db:"course"`
	StudentNumber string    `db:"student_number"`
}

// Teacher represents additional information for teacher users
type Teacher struct {
	UserID     uuid.UUID `db:"user_id"`
	FullName   string    `db:"full_name"`
	Department string    `db:"department"`
	Position   string    `db:"position"`
	TeacherID  string    `db:"teacher_id"`
}
