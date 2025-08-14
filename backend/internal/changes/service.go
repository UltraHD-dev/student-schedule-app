// Package changes реализует Change Detection Service
// В соответствии с ТЗ: "Change Detection Service - отслеживание изменений"
package changes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/schedule"
	"github.com/google/uuid"
)

// Service предоставляет функции для отслеживания изменений в расписании
type Service struct {
	scheduleRepo *schedule.Repository
}

// NewService создает новый сервис отслеживания изменений
func NewService(scheduleRepo *schedule.Repository) *Service {
	return &Service{
		scheduleRepo: scheduleRepo,
	}
}

// DetectChanges обнаруживает изменения в расписании
func (s *Service) DetectChanges(ctx context.Context) error {
	log.Println("Начинаем обнаружение изменений в расписании")

	// TODO: Реализовать логику обнаружения изменений
	// Это может включать:
	// 1. Получение последнего снапшота расписания
	// 2. Сравнение с текущими изменениями
	// 3. Определение типа изменений (замена, отмена, добавление)
	// 4. Создание записей в schedule_changes

	log.Println("Обнаружение изменений завершено")
	return nil
}

// ApplyChanges применяет обнаруженные изменения к актуальному расписанию
func (s *Service) ApplyChanges(ctx context.Context, changes []schedule.ScheduleChange) error {
	log.Println("Применяем изменения к актуальному расписанию")

	// TODO: Реализовать применение изменений:
	// 1. Обновление таблицы current_schedule
	// 2. Пометка изменений как примененных

	// Пока просто логируем количество изменений
	log.Printf("Получено %d изменений для применения", len(changes))

	log.Println("Изменения применены успешно")
	return nil
}

// GetChangesForGroup получает изменения для конкретной группы на определенную дату
func (s *Service) GetChangesForGroup(ctx context.Context, groupName string, date time.Time) ([]schedule.ScheduleChange, error) {
	// TODO: Реализовать получение изменений для группы
	// Пока возвращаем пустой список
	return []schedule.ScheduleChange{}, nil
}

// CreateChange создает новую запись об изменении
func (s *Service) CreateChange(ctx context.Context, change *schedule.ScheduleChange) error {
	// Генерируем ID если не задан
	if change.ID == uuid.Nil {
		change.ID = uuid.New()
	}

	// Устанавливаем время создания если не задано
	if change.CreatedAt.IsZero() {
		change.CreatedAt = time.Now()
	}

	// По умолчанию изменения активны
	if !change.IsActive {
		change.IsActive = true
	}

	err := s.scheduleRepo.CreateChange(ctx, change)
	if err != nil {
		return fmt.Errorf("ошибка создания записи об изменении: %w", err)
	}

	log.Printf("Создана запись об изменении: %s для группы %s", change.ID, change.GroupName)
	return nil
}
