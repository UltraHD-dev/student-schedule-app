// Package users предоставляет бизнес-логику для работы с пользователями
// В соответствии с ТЗ: "User Management Service - управление пользователями"
package users

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service предоставляет бизнес-логику для работы с пользователями
type Service struct {
	repo *Repository
}

// NewService создает новый сервис пользователей
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// RegisterUserInput содержит данные для регистрации нового пользователя
type RegisterUserInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     Role   `json:"role" validate:"required"`
}

// RegisterStudentInput содержит данные для регистрации студента
type RegisterStudentInput struct {
	RegisterUserInput
	GroupName     string `json:"group_name" validate:"required"`
	Faculty       string `json:"faculty"`
	Course        int    `json:"course" validate:"min=1,max=4"`
	StudentNumber string `json:"student_number"`
}

// RegisterTeacherInput содержит данные для регистрации преподавателя
type RegisterTeacherInput struct {
	RegisterUserInput
	FullName   string `json:"full_name" validate:"required"`
	Department string `json:"department"`
	Position   string `json:"position"`
	TeacherID  string `json:"teacher_id"`
}

// RegisterUser регистрирует нового пользователя
func (s *Service) RegisterUser(ctx context.Context, input RegisterUserInput) (*User, error) {
	// Проверяем, что пользователя с таким email еще нет
	_, err := s.repo.GetUserByEmail(ctx, input.Email)
	if err == nil {
		return nil, fmt.Errorf("user with email %s already exists", input.Email)
	}

	// Хэшируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Создаем пользователя
	user := &User{
		ID:       uuid.New(),
		Email:    input.Email,
		Password: string(hashedPassword),
		Role:     input.Role,
		IsActive: true,
	}

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// RegisterStudent регистрирует нового студента
func (s *Service) RegisterStudent(ctx context.Context, input RegisterStudentInput) (*User, *Student, error) {
	// Устанавливаем роль студента
	input.Role = RoleStudent

	// Регистрируем пользователя
	user, err := s.RegisterUser(ctx, input.RegisterUserInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register user: %w", err)
	}

	// Создаем профиль студента
	student := &Student{
		UserID:        user.ID,
		GroupName:     input.GroupName,
		Faculty:       input.Faculty,
		Course:        input.Course,
		StudentNumber: input.StudentNumber,
	}

	err = s.repo.CreateStudent(ctx, student)
	if err != nil {
		// Note: In a real application, you might want to rollback user creation here
		return nil, nil, fmt.Errorf("failed to create student profile: %w", err)
	}

	return user, student, nil
}

// RegisterTeacher регистрирует нового преподавателя
func (s *Service) RegisterTeacher(ctx context.Context, input RegisterTeacherInput) (*User, *Teacher, error) {
	// Устанавливаем роль преподавателя
	input.Role = RoleTeacher

	// Регистрируем пользователя
	user, err := s.RegisterUser(ctx, input.RegisterUserInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register user: %w", err)
	}

	// Создаем профиль преподавателя
	teacher := &Teacher{
		UserID:     user.ID,
		FullName:   input.FullName,
		Department: input.Department,
		Position:   input.Position,
		TeacherID:  input.TeacherID,
	}

	err = s.repo.CreateTeacher(ctx, teacher)
	if err != nil {
		// Note: In a real application, you might want to rollback user creation here
		return nil, nil, fmt.Errorf("failed to create teacher profile: %w", err)
	}

	return user, teacher, nil
}

// AuthenticateUser аутентифицирует пользователя по email и паролю
func (s *Service) AuthenticateUser(ctx context.Context, email, password string) (*User, error) {
	return s.repo.AuthenticateUser(ctx, email, password)
}

// GetUserByID получает пользователя по ID
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetUserByID(ctx, id)
}

// GetUserByEmail получает пользователя по email
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.repo.GetUserByEmail(ctx, email)
}
