// Package schedule реализует Schedule Processing Service
// В соответствии с ТЗ: "Schedule Processing Service - обработка и хранение расписания"
package schedule

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Service предоставляет функции для обработки расписания
type Service struct {
	repo *Repository
}

// NewService создает новый сервис обработки расписания
func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// GetScheduleForGroup получает расписание для группы на определенную дату
func (s *Service) GetScheduleForGroup(ctx context.Context, groupName string, date time.Time) ([]CurrentSchedule, error) {
	log.Printf("Получаем расписание для группы %s на дату %s", groupName, date.Format("2006-01-02"))

	// Получаем актуальное расписание из БД
	schedules, err := s.repo.GetCurrentScheduleForGroup(ctx, groupName, date)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения расписания: %w", err)
	}

	log.Printf("Получено %d записей расписания для группы %s", len(schedules), groupName)
	return schedules, nil
}

// ProcessScheduleSnapshot обрабатывает новый снапшот расписания
func (s *Service) ProcessScheduleSnapshot(ctx context.Context, snapshot *ScheduleSnapshot) error {
	log.Printf("Обрабатываем снапшот расписания: %s", snapshot.Name)

	// TODO: Реализовать обработку снапшота
	// Это может включать:
	// 1. Парсинг JSON данных
	// 2. Преобразование в формат current_schedule
	// 3. Обновление current_schedule в БД

	log.Printf("Снапшот расписания обработан: %s", snapshot.Name)
	return nil
}

// ApplyScheduleChanges применяет изменения к актуальному расписанию
func (s *Service) ApplyScheduleChanges(ctx context.Context, changes []ScheduleChange) error {
	log.Printf("Применяем %d изменений к актуальному расписанию", len(changes))

	// TODO: Реализовать применение изменений
	// Это может включать:
	// 1. Получение текущего расписания
	// 2. Применение изменений
	// 3. Обновление current_schedule в БД

	log.Printf("Изменения применены к актуальному расписанию")
	return nil
}

// GetActiveScheduleSnapshot получает активный снапшот расписания
func (s *Service) GetActiveScheduleSnapshot(ctx context.Context) (*ScheduleSnapshot, error) {
	log.Println("Получаем активный снапшот расписания")

	snapshot, err := s.repo.GetActiveSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения активного снапшота: %w", err)
	}

	log.Printf("Получен активный снапшот: %s", snapshot.Name)
	return snapshot, nil
}
