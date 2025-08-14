// Package jwt предоставляет функции для работы с JWT токенами
// В соответствии с требованием ТЗ: "JWT токены для аутентификации"
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims структура для хранения данных в JWT токене
// Содержит стандартные поля и дополнительную информацию о пользователе
type Claims struct {
	UserID               uuid.UUID `json:"user_id"` // Уникальный ID пользователя
	Email                string    `json:"email"`   // Email пользователя
	Role                 string    `json:"role"`    // Роль пользователя (student, teacher, admin)
	jwt.RegisteredClaims           // Встроенные стандартные поля JWT
}

// Manager отвечает за создание и проверку JWT токенов
type Manager struct {
	secretKey     []byte        // Секретный ключ для подписи токенов
	tokenLifetime time.Duration // Время жизни токена
}

// NewManager создает новый менеджер JWT
// secretKey - секретный ключ для подписи токенов
// lifetime - время жизни токена (например, 24 * time.Hour)
func NewManager(secretKey string, lifetime time.Duration) *Manager {
	return &Manager{
		secretKey:     []byte(secretKey),
		tokenLifetime: lifetime,
	}
}

// GenerateToken создает новый JWT токен для пользователя
// userID - уникальный ID пользователя
// email - email пользователя
// role - роль пользователя
// Возвращает строку токена и ошибку (если есть)
func (m *Manager) GenerateToken(userID uuid.UUID, email, role string) (string, error) {
	// Создаем claims (данные, которые будут в токене)
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			// Устанавливаем время истечения токена
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenLifetime)),
			// Устанавливаем время создания токена
			IssuedAt: jwt.NewNumericDate(time.Now()),
			// Уникальный идентификатор токена
			ID: uuid.New().String(),
		},
	}

	// Создаем токен с методом подписи HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен нашим секретным ключом
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("ошибка подписи токена: %w", err)
	}

	return tokenString, nil
}

// ParseToken проверяет и парсит JWT токен
// tokenString - строка токена для проверки
// Возвращает распарсенные claims и ошибку (если есть)
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	// Парсим токен
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неподдерживаемый метод подписи: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга токена: %w", err)
	}

	// Проверяем валидность токена
	if !token.Valid {
		return nil, fmt.Errorf("токен недействителен")
	}

	return claims, nil
}
