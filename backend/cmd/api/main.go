// Package main реализует основной API сервер с gRPC
// В соответствии с ТЗ: "API Gateway - gRPC API"
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/config"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/grpc"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/jwt"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/users"
	_ "github.com/lib/pq"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", cfg.GetDatabaseDSN())
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Ошибка закрытия соединения с БД: %v", err)
		}
	}()

	// Проверяем подключение к БД
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Ошибка проверки подключения к БД: %v", err)
	}

	log.Println("Успешное подключение к базе данных")

	// Инициализируем компоненты
	userRepo := users.NewRepository(db)
	userService := users.NewService(userRepo)
	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.GetJWTTokenLifetime())

	// Инициализируем gRPC сервер
	grpcServer := grpc.NewServer(userService, jwtManager)

	// Запускаем gRPC сервер в отдельной горутине
	go func() {
		if err := grpcServer.Start(50051); err != nil {
			log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
		}
	}()

	log.Println("gRPC API Gateway запущен на порту 50051")
	log.Println("Доступные сервисы:")
	log.Println("  UserService:")
	log.Println("    - RegisterStudent")
	log.Println("    - RegisterTeacher")
	log.Println("    - Login")
	log.Println("    - GetProfile")

	// Ожидаем сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Получен сигнал завершения, останавливаем сервер...")

	log.Println("Сервер остановлен")
}
