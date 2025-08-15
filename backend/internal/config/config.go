// Package config предоставляет функции для работы с конфигурацией приложения
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config основная структура конфигурации приложения
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Scraper  ScraperConfig  `yaml:"scraper"`
	JWT      JWTConfig      `yaml:"jwt"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Port int `yaml:"port"`
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	Addr string `yaml:"addr"`
}

// ScraperConfig конфигурация для scraper сервиса
type ScraperConfig struct {
	BaseURL          string        `yaml:"base_url"`
	Timeout          time.Duration `yaml:"timeout"`
	MainScheduleGIDs []int64       `yaml:"main_schedule_gids"` // Список gid листов основного расписания
	ChangesGID       int64         `yaml:"changes_gid"`        // gid листа изменений
}

// JWTConfig конфигурация JWT
type JWTConfig struct {
	Secret     string        `yaml:"secret"`
	Expiration time.Duration `yaml:"expiration"`
}

// LoadConfig загружает конфигурацию из YAML файла
func LoadConfig(filename string) (*Config, error) {
	// Открываем файл конфигурации
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", filename, err)
	}
	defer file.Close()

	// Создаем экземпляр конфигурации
	cfg := &Config{}

	// Декодируем YAML в структуру
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file %s: %w", filename, err)
	}

	// Устанавливаем значения по умолчанию, если они не заданы
	if cfg.Scraper.Timeout == 0 {
		cfg.Scraper.Timeout = 30 * time.Second
	}

	return cfg, nil
}
