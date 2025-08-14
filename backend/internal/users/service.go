package users

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service provides user business logic
type Service struct {
	repo *Repository
}

// NewService creates a new user service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// RegisterUserInput contains data needed to register a new user
type RegisterUserInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     Role   `json:"role" validate:"required"`
}

// RegisterStudentInput contains data needed to register a student
type RegisterStudentInput struct {
	RegisterUserInput
	GroupName     string `json:"group_name" validate:"required"`
	Faculty       string `json:"faculty"`
	Course        int    `json:"course" validate:"min=1,max=4"`
	StudentNumber string `json:"student_number"`
}

// RegisterTeacherInput contains data needed to register a teacher
type RegisterTeacherInput struct {
	RegisterUserInput
	FullName   string `json:"full_name" validate:"required"`
	Department string `json:"department"`
	Position   string `json:"position"`
	TeacherID  string `json:"teacher_id"`
}

// RegisterUser registers a new user
func (s *Service) RegisterUser(ctx context.Context, input RegisterUserInput) (*User, error) {
	// Check if user already exists
	_, err := s.repo.GetUserByEmail(ctx, input.Email)
	if err == nil {
		return nil, fmt.Errorf("user with email %s already exists", input.Email)
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
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

// RegisterStudent registers a new student
func (s *Service) RegisterStudent(ctx context.Context, input RegisterStudentInput) (*User, *Student, error) {
	// Set role to student
	input.Role = RoleStudent

	// Register user
	user, err := s.RegisterUser(ctx, input.RegisterUserInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register user: %w", err)
	}

	// Create student profile
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

// RegisterTeacher registers a new teacher
func (s *Service) RegisterTeacher(ctx context.Context, input RegisterTeacherInput) (*User, *Teacher, error) {
	// Set role to teacher
	input.Role = RoleTeacher

	// Register user
	user, err := s.RegisterUser(ctx, input.RegisterUserInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register user: %w", err)
	}

	// Create teacher profile
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

// AuthenticateUser authenticates a user by email and password
func (s *Service) AuthenticateUser(ctx context.Context, email, password string) (*User, error) {
	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is deactivated")
	}

	// Compare password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil

}

// GetUserByID retrieves a user by ID
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetUserByID(ctx, id)
}
