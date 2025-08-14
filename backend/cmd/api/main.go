// Package main реализует основной API сервер с gRPC
// В соответствии с ТЗ: "API Gateway - gRPC API"
package main

import (
	"context"
	"database/sql"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/changes"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/config"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/grpc"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/jwt"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/notifications"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/schedule"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/scraper"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/users"
	_ "github.com/lib/pq"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	// Инициализируем schedule репозиторий и сервис
	scheduleRepo := schedule.NewRepository(db)
	scheduleService := schedule.NewService(scheduleRepo)

	// Инициализируем notification репозиторий и сервис
	notificationRepo := notifications.NewRepository(db)
	notificationService := notifications.NewService(userRepo, scheduleRepo, notificationRepo)

	// Инициализируем change detection сервис
	changeService := changes.NewService(scheduleRepo)

	// Инициализируем scraper сервис с передачей notification и change сервисов
	scraperConfig := scraper.Config{
		BaseURL: "https://kcpt72.ru/schedule/",
		Timeout: 30 * time.Second,
	}
	scraperService := scraper.NewService(scraperConfig, scheduleRepo, notificationService, changeService)

	// Передаем notificationService в scraperService
	// TODO: Реализовать передачу notificationService в scraperService

	// Инициализируем gRPC сервер
	grpcServer := grpc.NewServer(userService, jwtManager)

	// Запускаем gRPC сервер в отдельной горутине
	go func() {
		if err := grpcServer.Start(50051, scheduleService, userService); err != nil { // Передаем scheduleService
			log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
		}
	}()

	// Запускаем периодический парсинг в отдельной горутине
	scraperCtx, scraperCancel := context.WithCancel(context.Background())
	go scraperService.StartPeriodicScraping(scraperCtx)

	log.Println("gRPC API Gateway запущен на порту 50051")
	log.Println("Web Scraper Service запущен")
	log.Println("Change Detection Service запущен")
	log.Println("Notification Service запущен")
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

	// Отменяем контекст для scraper сервиса
	scraperCancel()

	log.Println("Сервер остановлен")
}
