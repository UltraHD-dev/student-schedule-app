// Package users предоставляет доступ к хранению пользователей
// В соответствии с ТЗ: "User Management Service - управление пользователями"
package users

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Repository предоставляет доступ к хранению пользователей
type Repository struct {
	db *sql.DB
}

// NewRepository создает новый репозиторий пользователей
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser создает нового пользователя в базе данных
func (r *Repository) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, email, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at`

	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query, user.ID, user.Email, user.Password, user.Role, user.IsActive).
		Scan(&createdAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.CreatedAt = createdAt
	return nil
}

// GetUserByEmail получает пользователя по email
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, created_at, last_login, is_active
		FROM users
		WHERE email = $1`

	user := &User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.LastLogin,
		&user.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetUserByID получает пользователя по ID
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, created_at, last_login, is_active
		FROM users
		WHERE id = $1`

	user := &User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.LastLogin,
		&user.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// CreateStudent создает профиль студента
func (r *Repository) CreateStudent(ctx context.Context, student *Student) error {
	query := `
		INSERT INTO students (user_id, group_name, faculty, course, student_number)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.ExecContext(ctx, query, student.UserID, student.GroupName, student.Faculty, student.Course, student.StudentNumber)
	if err != nil {
		return fmt.Errorf("failed to create student profile: %w", err)
	}

	return nil
}

// CreateTeacher создает профиль преподавателя
func (r *Repository) CreateTeacher(ctx context.Context, teacher *Teacher) error {
	query := `
		INSERT INTO teachers (user_id, full_name, department, position, teacher_id)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.ExecContext(ctx, query, teacher.UserID, teacher.FullName, teacher.Department, teacher.Position, teacher.TeacherID)
	if err != nil {
		return fmt.Errorf("failed to create teacher profile: %w", err)
	}

	return nil
}

// GetStudentsByGroup получает всех студентов определенной группы
func (r *Repository) GetStudentsByGroup(ctx context.Context, groupName string) ([]uuid.UUID, error) {
	query := `
		SELECT s.user_id
		FROM students s
		JOIN users u ON s.user_id = u.id
		WHERE s.group_name = $1 AND u.is_active = true`

	rows, err := r.db.QueryContext(ctx, query, groupName)
	if err != nil {
		return nil, fmt.Errorf("failed to get students by group: %w", err)
	}
	defer rows.Close()

	var studentIDs []uuid.UUID
	for rows.Next() {
		var studentID uuid.UUID
		err := rows.Scan(&studentID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan student ID: %w", err)
		}
		studentIDs = append(studentIDs, studentID)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return studentIDs, nil
}

// AuthenticateUser аутентифицирует пользователя по email и паролю
func (r *Repository) AuthenticateUser(ctx context.Context, email, password string) (*User, error) {
	// Получаем пользователя по email
	user, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Проверяем, что пользователь активен
	if !user.IsActive {
		return nil, fmt.Errorf("user account is deactivated")
	}

	// Сравниваем хэш пароля
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}
