// Package handlers предоставляет HTTP handlers для работы с пользователями
// В соответствии с ТЗ, но в дальнейшем будет заменен на gRPC API Gateway
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/auth"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/jwt"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/users"
)

// AuthHandler обрабатывает HTTP запросы, связанные с аутентификацией
type AuthHandler struct {
	userService *users.Service
	jwtManager  *jwt.Manager
}

// NewAuthHandler создает новый handler для аутентификации
func NewAuthHandler(userService *users.Service, jwtManager *jwt.Manager) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		jwtManager:  jwtManager,
	}
}

// RegisterRequest структура для данных регистрации из тела запроса
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"required,oneof=student teacher"`

	// Поля для студентов
	GroupName     string `json:"group_name,omitempty"`
	Faculty       string `json:"faculty,omitempty"`
	Course        int    `json:"course,omitempty"`
	StudentNumber string `json:"student_number,omitempty"`

	// Поля для преподавателей
	FullName   string `json:"full_name,omitempty"`
	Department string `json:"department,omitempty"`
	Position   string `json:"position,omitempty"`
	TeacherID  string `json:"teacher_id,omitempty"`
}

// RegisterResponse структура для ответа на запрос регистрации
type RegisterResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	User    interface{} `json:"user,omitempty"`
	Profile interface{} `json:"profile,omitempty"`
}

// Register обрабатывает регистрацию новых пользователей
// POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	// Декодируем тело запроса в структуру
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат данных в запросе", http.StatusBadRequest)
		return
	}

	// Валидируем обязательные поля в зависимости от роли
	switch req.Role {
	case "student":
		if req.GroupName == "" {
			http.Error(w, "Для студентов обязательно указание группы (group_name)", http.StatusBadRequest)
			return
		}
	case "teacher":
		if req.FullName == "" {
			http.Error(w, "Для преподавателей обязательно указание ФИО (full_name)", http.StatusBadRequest)
			return
		}
	}

	// Подготавливаем данные для регистрации
	registerInput := users.RegisterUserInput{
		Email:    req.Email,
		Password: req.Password,
	}

	// В зависимости от роли регистрируем студента или преподавателя
	switch req.Role {
	case "student":
		studentInput := users.RegisterStudentInput{
			RegisterUserInput: registerInput,
			GroupName:         req.GroupName,
			Faculty:           req.Faculty,
			Course:            req.Course,
			StudentNumber:     req.StudentNumber,
		}

		// Устанавливаем роль
		studentInput.Role = users.RoleStudent

		// Регистрируем студента
		user, student, err := h.userService.RegisterStudent(r.Context(), studentInput)
		if err != nil {
			log.Printf("Ошибка регистрации студента: %v", err)
			http.Error(w, fmt.Sprintf("Ошибка регистрации: %v", err), http.StatusInternalServerError)
			return
		}

		// Формируем успешный ответ
		response := RegisterResponse{
			Success: true,
			Message: "Студент успешно зарегистрирован",
			User: map[string]interface{}{
				"id":    user.ID,
				"email": user.Email,
				"role":  user.Role,
			},
			Profile: map[string]interface{}{
				"user_id":        student.UserID,
				"group_name":     student.GroupName,
				"faculty":        student.Faculty,
				"course":         student.Course,
				"student_number": student.StudentNumber,
			},
		}

		// Отправляем ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

	case "teacher":
		teacherInput := users.RegisterTeacherInput{
			RegisterUserInput: registerInput,
			FullName:          req.FullName,
			Department:        req.Department,
			Position:          req.Position,
			TeacherID:         req.TeacherID,
		}

		// Устанавливаем роль
		teacherInput.Role = users.RoleTeacher

		// Регистрируем преподавателя
		user, teacher, err := h.userService.RegisterTeacher(r.Context(), teacherInput)
		if err != nil {
			log.Printf("Ошибка регистрации преподавателя: %v", err)
			http.Error(w, fmt.Sprintf("Ошибка регистрации: %v", err), http.StatusInternalServerError)
			return
		}

		// Формируем успешный ответ
		response := RegisterResponse{
			Success: true,
			Message: "Преподаватель успешно зарегистрирован",
			User: map[string]interface{}{
				"id":    user.ID,
				"email": user.Email,
				"role":  user.Role,
			},
			Profile: map[string]interface{}{
				"user_id":    teacher.UserID,
				"full_name":  teacher.FullName,
				"department": teacher.Department,
				"position":   teacher.Position,
				"teacher_id": teacher.TeacherID,
			},
		}

		// Отправляем ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Неверная роль. Допустимые значения: 'student', 'teacher'", http.StatusBadRequest)
	}
}

// LoginRequest структура для данных входа из тела запроса
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse структура для ответа на запрос входа
type LoginResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Token   string      `json:"token,omitempty"`
	User    interface{} `json:"user,omitempty"`
}

// Login обрабатывает вход пользователя в систему
// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	// Декодируем тело запроса в структуру
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат данных в запросе", http.StatusBadRequest)
		return
	}

	// Аутентифицируем пользователя
	user, err := h.userService.AuthenticateUser(r.Context(), req.Email, req.Password)
	if err != nil {
		log.Printf("Ошибка аутентификации пользователя %s: %v", req.Email, err)
		http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
		return
	}

	// Генерируем JWT токен
	token, err := h.jwtManager.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		log.Printf("Ошибка генерации JWT токена для пользователя %s: %v", user.Email, err)
		http.Error(w, "Ошибка генерации токена", http.StatusInternalServerError)
		return
	}

	// Формируем успешный ответ с токеном
	response := LoginResponse{
		Success: true,
		Message: "Вход выполнен успешно",
		Token:   token,
		User: map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"role":       user.Role,
			"created_at": user.CreatedAt,
			"is_active":  user.IsActive,
		},
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ProfileResponse структура для ответа с профилем пользователя
type ProfileResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	User    interface{} `json:"user"`
	Profile interface{} `json:"profile,omitempty"`
}

// Profile обрабатывает запрос на получение профиля текущего пользователя
// GET /api/v1/auth/profile
// Требует аутентификации
func (h *AuthHandler) Profile(w http.ResponseWriter, r *http.Request) {
	// Получаем информацию о пользователе из контекста (добавлена middleware)
	userInfo, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "Ошибка получения информации о пользователе", http.StatusInternalServerError)
		return
	}

	// Получаем полную информацию о пользователе из БД
	user, err := h.userService.GetUserByID(r.Context(), userInfo.ID)
	if err != nil {
		log.Printf("Ошибка получения пользователя %s: %v", userInfo.ID, err)
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Формируем ответ с базовой информацией о пользователе
	response := ProfileResponse{
		Success: true,
		Message: "Профиль получен успешно",
		User: map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"role":       user.Role,
			"created_at": user.CreatedAt,
			"is_active":  user.IsActive,
		},
	}

	// В зависимости от роли добавляем дополнительную информацию
	switch user.Role {
	case users.RoleStudent:
		// Для студентов получаем информацию из таблицы students
		// TODO: Реализовать получение профиля студента
		response.Profile = map[string]interface{}{
			"message": "Профиль студента будет реализован позже",
		}
	case users.RoleTeacher:
		// Для преподавателей получаем информацию из таблицы teachers
		// TODO: Реализовать получение профиля преподавателя
		response.Profile = map[string]interface{}{
			"message": "Профиль преподавателя будет реализован позже",
		}
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
