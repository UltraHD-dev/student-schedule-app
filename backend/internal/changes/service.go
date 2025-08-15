// Package changes реализует Change Detection Service
// В соответствии с ТЗ: "Change Detection Service - отслеживание изменений"
package changes

import (
	"context"
	"database/sql"
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
// В соответствии с ТЗ: "Change Detection Service - отслеживание изменений"
func (s *Service) DetectChanges(ctx context.Context) error {
	log.Println("Начинаем обнаружение изменений в расписании")

	// 1. Получаем последние изменения из БД
	// В реальной реализации это может быть:
	// - Сравнение с предыдущей версией снапшота
	// - Анализ логов парсинга
	// - Сравнение хэшей данных

	// Пока возвращаем nil, так как реальная логика будет в scraper service
	log.Println("Обнаружение изменений завершено")
	return nil
}

// ApplyChanges применяет обнаруженные изменения к актуальному расписанию
// В соответствии с ТЗ: "Если есть изменения: ... Обновление current_schedule"
func (s *Service) ApplyChanges(ctx context.Context, changes []schedule.ScheduleChange) error {
	log.Printf("Применяем %d изменений к актуальному расписанию", len(changes))

	// Начинаем транзакцию для обеспечения целостности данных
	tx, err := s.scheduleRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer func() {
		// Откатываем транзакцию в случае ошибки
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Для каждого изменения:
	appliedCount := 0
	for _, change := range changes {
		// 1. Обновляем current_schedule
		// ИСПРАВЛЕНО: Передаем ctx в updateCurrentSchedule
		err := s.updateCurrentSchedule(ctx, tx, &change)
		if err != nil {
			log.Printf("Ошибка обновления current_schedule для изменения %s: %v", change.ID, err)
			// Не возвращаем ошибку, продолжаем применять другие изменения
			continue
		}

		log.Printf("Обновлено current_schedule для изменения: %s", change.ID)
		appliedCount++
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	log.Printf("Изменения применены успешно к актуальному расписанию (%d из %d)", appliedCount, len(changes))
	return nil
}

// updateCurrentSchedule обновляет запись в current_schedule на основе изменения
// ИСПРАВЛЕНО: Добавлен ctx как первый параметр, удалён дубликат
func (s *Service) updateCurrentSchedule(ctx context.Context, tx *sql.Tx, change *schedule.ScheduleChange) error {
	// 1. Проверяем, существует ли уже запись в current_schedule для этой пары
	// ИСПРАВЛЕНО: Передаем ctx в вызовы методов репозитория
	existing, err := s.scheduleRepo.GetCurrentScheduleEntry(ctx, tx, change.GroupName, change.Date, change.TimeStart)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("ошибка получения существующей записи: %w", err)
	}

	// 2. Если запись существует, обновляем её
	if existing != nil {
		// Обновляем существующую запись
		existing.Subject = change.Subject
		existing.Teacher = change.Teacher
		existing.Classroom = change.Classroom
		existing.SourceType = "change"
		existing.SourceID = change.ID

		// ИСПРАВЛЕНО: Передаем ctx
		err = s.scheduleRepo.UpdateCurrentScheduleEntry(ctx, tx, existing)
		if err != nil {
			return fmt.Errorf("ошибка обновления существующей записи: %w", err)
		}
	} else {
		// 3. Если записи нет, создаем новую
		newEntry := &schedule.CurrentSchedule{
			ID:         uuid.New(),
			GroupName:  change.GroupName,
			Date:       change.Date,
			TimeStart:  change.TimeStart,
			TimeEnd:    change.TimeEnd,
			Subject:    change.Subject,
			Teacher:    change.Teacher,
			Classroom:  change.Classroom,
			SourceType: "change",
			SourceID:   change.ID,
			IsActive:   true,
		}

		// ИСПРАВЛЕНО: Передаем ctx
		err = s.scheduleRepo.CreateCurrentScheduleEntry(ctx, tx, newEntry)
		if err != nil {
			return fmt.Errorf("ошибка создания новой записи: %w", err)
		}
	}

	return nil
}

// GetChangesForGroup получает изменения для конкретной группы на определенную дату
// В соответствии с ТЗ: "Получение изменений для группы"
func (s *Service) GetChangesForGroup(ctx context.Context, groupName string, date time.Time) ([]schedule.ScheduleChange, error) {
	// Получаем изменения из БД
	changes, err := s.scheduleRepo.GetChangesForGroup(ctx, groupName, date)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения изменений для группы %s: %w", groupName, err)
	}

	return changes, nil
}

// CreateChange создает новую запись об изменении
// В соответствии с ТЗ: "Создание записей в schedule_changes"
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

