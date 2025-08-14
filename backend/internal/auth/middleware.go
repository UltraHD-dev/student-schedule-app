// Package auth предоставляет функции для аутентификации и авторизации
// В соответствии с требованием ТЗ о JWT-аутентификации
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/jwt"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/users"
	"github.com/google/uuid"
)

// Ключи для хранения данных в контексте HTTP запроса
type contextKey string

const (
	// Ключ для хранения информации о пользователе в контексте
	UserContextKey contextKey = "user"
)

// UserFromContext извлекает информацию о пользователе из контекста HTTP запроса
func UserFromContext(ctx context.Context) (*UserInfo, bool) {
	user, ok := ctx.Value(UserContextKey).(*UserInfo)
	return user, ok
}

// UserInfo содержит информацию о пользователе, извлеченную из JWT токена
type UserInfo struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}

// Middleware предоставляет middleware функции для аутентификации
type Middleware struct {
	jwtManager *jwt.Manager
	userRepo   *users.Repository
}

// NewMiddleware создает новый middleware для аутентификации
func NewMiddleware(jwtManager *jwt.Manager, userRepo *users.Repository) *Middleware {
	return &Middleware{
		jwtManager: jwtManager,
		userRepo:   userRepo,
	}
}

// Authenticate проверяет JWT токен из заголовка Authorization
// и добавляет информацию о пользователе в контекст запроса
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем заголовок Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Требуется аутентификация: отсутствует заголовок Authorization", http.StatusUnauthorized)
			return
		}

		// Проверяем формат заголовка (должен начинаться с "Bearer ")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Неверный формат токена: должен начинаться с 'Bearer '", http.StatusUnauthorized)
			return
		}

		// Извлекаем токен (убираем "Bearer " в начале)
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Парсим и проверяем токен
		claims, err := m.jwtManager.ParseToken(tokenString)
		if err != nil {
			http.Error(w, fmt.Sprintf("Неверный токен: %v", err), http.StatusUnauthorized)
			return
		}

		// Проверяем, что пользователь еще существует и активен
		user, err := m.userRepo.GetUserByID(r.Context(), claims.UserID)
		if err != nil {
			http.Error(w, "Пользователь не найден", http.StatusUnauthorized)
			return
		}

		if !user.IsActive {
			http.Error(w, "Пользователь деактивирован", http.StatusUnauthorized)
			return
		}

		// Создаем объект с информацией о пользователе
		userInfo := &UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Role:  string(user.Role),
		}

		// Добавляем информацию о пользователе в контекст запроса
		ctx := context.WithValue(r.Context(), UserContextKey, userInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole проверяет, что у пользователя есть одна из требуемых ролей
func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем информацию о пользователе из контекста
			user, ok := UserFromContext(r.Context())
			if !ok {
				http.Error(w, "Требуется аутентификация", http.StatusUnauthorized)
				return
			}

			// Проверяем, есть ли у пользователя одна из требуемых ролей
			hasRole := false
			for _, role := range roles {
				if user.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, "Доступ запрещен: недостаточно прав", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
