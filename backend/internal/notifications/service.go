// Package notifications реализует Notification Service
// В соответствии с ТЗ: "Notification Service - система уведомлений"
package notifications

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/schedule"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/users"
	"github.com/google/uuid"
)

// Service предоставляет функции для отправки уведомлений
type Service struct {
	userRepo         *users.Repository    // Добавляем userRepo
	scheduleRepo     *schedule.Repository // Оставляем scheduleRepo
	notificationRepo *Repository          // Добавляем notificationRepo
}

// NotificationType тип уведомления
type NotificationType string

const (
	NotificationTypeScheduleChange NotificationType = "schedule_change"
	NotificationTypeSystem         NotificationType = "system"
	NotificationTypeImportant      NotificationType = "important"
)

// NotificationChangeType тип изменения в уведомлении
type NotificationChangeType string

const (
	NotificationChangeTypeReplacement  NotificationChangeType = "replacement"
	NotificationChangeTypeCancellation NotificationChangeType = "cancellation"
	NotificationChangeTypeAddition     NotificationChangeType = "addition"
)

// NewService создает новый сервис уведомлений
func NewService(userRepo *users.Repository, scheduleRepo *schedule.Repository, notificationRepo *Repository) *Service {
	return &Service{
		userRepo:         userRepo,
		scheduleRepo:     scheduleRepo,
		notificationRepo: notificationRepo,
	}
}

// Notification представляет уведомление для пользователя
type Notification struct {
	ID           uuid.UUID        `db:"id"`
	UserID       uuid.UUID        `db:"user_id"`
	Title        string           `db:"title"`
	Message      string           `db:"message"`
	Type         NotificationType `db:"type"`
	RelatedGroup string           `db:"related_group"`
	RelatedDate  time.Time        `db:"related_date"`
	IsRead       bool             `db:"is_read"`
	CreatedAt    time.Time        `db:"created_at"`
}

// SendScheduleChangeNotification отправляет уведомление об изменении в расписании
func (s *Service) SendScheduleChangeNotification(ctx context.Context, change *schedule.ScheduleChange) error {
	log.Printf("Отправляем уведомление об изменении в расписании для группы %s", change.GroupName)

	// 1. Формируем сообщение уведомления в зависимости от типа изменения
	title, message := s.formatChangeMessage(change)

	// 2. Получаем всех студентов группы
	studentIDs, err := s.userRepo.GetStudentsByGroup(ctx, change.GroupName)
	if err != nil {
		return fmt.Errorf("ошибка получения студентов группы %s: %w", change.GroupName, err)
	}

	// Если нет студентов, выходим
	if len(studentIDs) == 0 {
		log.Printf("Нет студентов в группе %s для отправки уведомления", change.GroupName)
		return nil
	}

	// 3. Создаем уведомления для каждого студента
	var notificationErrors []error
	for _, studentID := range studentIDs {
		notification := &Notification{
			ID:           uuid.New(),
			UserID:       studentID,
			Title:        title,
			Message:      message,
			Type:         NotificationTypeScheduleChange,
			RelatedGroup: change.GroupName,
			RelatedDate:  change.Date,
			IsRead:       false,
		}

		// Создаем уведомление в БД
		err := s.notificationRepo.CreateNotification(ctx, notification)
		if err != nil {
			notificationErrors = append(notificationErrors, fmt.Errorf("ошибка создания уведомления для студента %s: %w", studentID, err))
			continue
		}

		log.Printf("Создано уведомление для студента %s: %s", studentID, title)
	}

	if len(notificationErrors) > 0 {
		// Возвращаем первую ошибку, если были ошибки
		return fmt.Errorf("ошибки при создании уведомлений: %v", notificationErrors[0])
	}

	log.Printf("Уведомление об изменении отправлено для группы %s (%d студентов)", change.GroupName, len(studentIDs))
	return nil
}

// SendNewScheduleNotification отправляет уведомление о новом основном расписании
func (s *Service) SendNewScheduleNotification(ctx context.Context, snapshot *schedule.ScheduleSnapshot) error {
	log.Println("Отправляем уведомление о новом основном расписании")

	title := "Обновлено расписание"
	message := fmt.Sprintf("Доступно новое расписание на период с %s по %s",
		snapshot.PeriodStart.Format("02.01.2006"),
		snapshot.PeriodEnd.Format("02.01.2006"))

	// TODO: Получить всех студентов и преподавателей
	// TODO: Создать уведомления для каждого

	log.Printf("Уведомление: %s - %s", title, message)
	log.Println("Уведомление о новом расписании отправлено")
	return nil
}

// formatChangeMessage форматирует сообщение уведомления об изменении
func (s *Service) formatChangeMessage(change *schedule.ScheduleChange) (string, string) {
	title := fmt.Sprintf("Изменения в расписании на %s", change.Date.Format("02.01.2006"))

	var message string
	switch change.ChangeType {
	case "replacement":
		message = fmt.Sprintf("Ваша пара по %s (%s) перенесена с %s на %s. Новый кабинет: %s",
			change.Subject, change.Teacher, change.OriginalSubject, change.TimeStart, change.Classroom)
	case "cancellation":
		message = fmt.Sprintf("Пара по %s (%s) в %s отменена",
			change.Subject, change.Teacher, change.TimeStart)
	case "addition":
		message = fmt.Sprintf("Добавлена новая пара по %s (%s) в %s. Кабинет: %s",
			change.Subject, change.Teacher, change.TimeStart, change.Classroom)
	default:
		message = fmt.Sprintf("Изменения в расписании: %s (%s) в %s. Кабинет: %s",
			change.Subject, change.Teacher, change.TimeStart, change.Classroom)
	}

	return title, message
}

// GetUnreadNotifications получает непрочитанные уведомления для пользователя
func (s *Service) GetUnreadNotifications(ctx context.Context, userID uuid.UUID) ([]Notification, error) {
	// TODO: Реализовать получение непрочитанных уведомлений из БД
	return []Notification{}, nil
}

// MarkAsRead помечает уведомление как прочитанное
func (s *Service) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	// TODO: Реализовать пометку уведомления как прочитанного в БД
	return nil
}
