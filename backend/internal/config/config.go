// Package config предоставляет функции для работы с конфигурацией приложения
// В соответствии с ТЗ: использование конфигурации
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config структура для хранения конфигурации приложения
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// JWTConfig конфигурация JWT
type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	TokenLifetimeHours int    `mapstructure:"expiration_hours"`
}

// LoadConfig загружает конфигурацию из файла или переменных окружения
func LoadConfig(path string) (*Config, error) {
	// Устанавливаем значения по умолчанию
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "student_user")
	viper.SetDefault("database.password", "student_pass")
	viper.SetDefault("database.dbname", "student_schedule_dev")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("jwt.secret", "your-super-secret-jwt-key-change-in-production")
	viper.SetDefault("jwt.expiration_hours", 24)

	// Устанавливаем путь к конфигурационным файлам
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Читаем конфигурацию
	if err := viper.ReadInConfig(); err != nil {
		// Если конфигурационный файл не найден, используем значения по умолчанию
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("ошибка чтения конфигурационного файла: %w", err)
		}
		fmt.Println("Конфигурационный файл не найден, используем значения по умолчанию")
	}

	// Привязываем переменные окружения
	viper.AutomaticEnv()

	// Создаем структуру конфигурации
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("ошибка парсинга конфигурации: %w", err)
	}

	return &config, nil
}

// GetDatabaseDSN возвращает строку подключения к базе данных
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

// GetJWTTokenLifetime возвращает время жизни JWT токена
func (c *Config) GetJWTTokenLifetime() time.Duration {
	return time.Duration(c.JWT.TokenLifetimeHours) * time.Hour
}
