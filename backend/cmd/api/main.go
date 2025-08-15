// Package main реализует основной API сервер с gRPC
// В соответствии с ТЗ: "API Gateway - gRPC API"
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/changes"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/config"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/grpc"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/jwt"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/notifications"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/schedule"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/scraper"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/users"
	_ "github.com/lib/pq"
)

func main() {
	// Загружаем конфигурацию
	// ИСПРАВЛЕНО: Указываем путь к конкретному файлу конфигурации
	cfg, err := config.LoadConfig("./configs/config.yaml")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// ИСПРАВЛЕНО: Формируем DSN вручную из полей конфигурации
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", dsn)
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

	// ИСПРАВЛЕНО: Используем cfg.JWT.Expiration вместо cfg.GetJWTTokenLifetime()
	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Expiration)

	// Инициализируем schedule репозиторий и сервис
	scheduleRepo := schedule.NewRepository(db)
	scheduleService := schedule.NewService(scheduleRepo)

	// Инициализируем notification репозиторий и сервис
	notificationRepo := notifications.NewRepository(db)
	notificationService := notifications.NewService(userRepo, scheduleRepo, notificationRepo)

	// Инициализируем change detection сервис
	changeService := changes.NewService(scheduleRepo)

	// Создание scraper сервиса
	scraperConfig := scraper.Config{
		BaseURL:          cfg.Scraper.BaseURL,
		Timeout:          cfg.Scraper.Timeout,
		MainScheduleGIDs: cfg.Scraper.MainScheduleGIDs, // Передаем список gid
		ChangesGID:       cfg.Scraper.ChangesGID,       // Передаем gid изменений
	}

	scraperService := scraper.NewService(scraperConfig, scheduleRepo, notificationService, changeService)

	// Инициализируем gRPC сервер
	grpcServer := grpc.NewServer(userService, jwtManager)

	// Запускаем gRPC сервер в отдельной горутине
	go func() {
		if err := grpcServer.Start(cfg.Server.Port, scheduleService, userService); err != nil {
			log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
		}
	}()

	// Немедленный запуск парсинга при старте сервера
	// В соответствии с ТЗ: "Немедленный запуск парсинга"
	log.Println("Немедленный запуск парсинга при старте сервера")

	// Создаем контекст для немедленного парсинга
	immediateCtx, immediateCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer immediateCancel()

	// Запускаем немедленный парсинг основного расписания
	if err := scraperService.ScrapeMainSchedule(immediateCtx); err != nil {
		log.Printf("Ошибка при немедленном парсинге основного расписания: %v", err)
	}

	// Запускаем немедленный парсинг изменений в расписании
	if err := scraperService.ScrapeScheduleChanges(immediateCtx); err != nil {
		log.Printf("Ошибка при немедленном парсинге изменений в расписании: %v", err)
	}

	// Запускаем периодический парсинг в отдельной горутине
	scraperCtx, scraperCancel := context.WithCancel(context.Background())
	go scraperService.StartPeriodicScraping(scraperCtx)

	log.Printf("gRPC API Gateway запущен на порту %d", cfg.Server.Port)
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
