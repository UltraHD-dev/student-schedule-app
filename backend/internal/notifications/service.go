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
	userRepo         *users.Repository
	scheduleRepo     *schedule.Repository
	notificationRepo *Repository
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
// В соответствии с ТЗ: "Отправка уведомлений"
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
			CreatedAt:    time.Now(),
		}

		// Создаем уведомление в БД
		err := s.notificationRepo.CreateNotification(ctx, notification)
		if err != nil {
			notificationErrors = append(notificationErrors, fmt.Errorf("ошибка создания уведомления для студента %s: %w", studentID, err))
			continue
		}

		log.Printf("Создано уведомление для студента %s: %s", studentID, title)

		// Отправляем push-уведомление
		if err := s.sendPushNotification(ctx, notification); err != nil {
			log.Printf("Ошибка отправки push уведомления студенту %s: %v", studentID, err)
		}
	}

	if len(notificationErrors) > 0 {
		// Возвращаем первую ошибку, если были ошибки
		return fmt.Errorf("ошибки при создании уведомлений: %v", notificationErrors[0])
	}

	log.Printf("Уведомление об изменении отправлено для группы %s (%d студентов)", change.GroupName, len(studentIDs))
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

// sendPushNotification отправляет push-уведомление
// В соответствии с ТЗ: "Получение уведомлений об изменениях"
func (s *Service) sendPushNotification(ctx context.Context, notification *Notification) error {
	// TODO: Здесь будет реальная логика отправки push-уведомлений
	// Например, с использованием FCM (Firebase Cloud Messaging) или APNs (Apple Push Notification Service)

	// Пока просто логируем отправку
	log.Printf("Отправка push уведомления пользователю %s: %s - %s",
		notification.UserID, notification.Title, notification.Message)

	// В реальной реализации здесь будет код для отправки через FCM/APNs
	// Например:
	// fcmClient := s.getFCMClient()
	// err := fcmClient.SendMessageToDevice(deviceToken, &fcm.Message{
	//     Title: notification.Title,
	//     Body:  notification.Message,
	//     Data: map[string]string{
	//         "notification_id": notification.ID.String(),
	//         "type":          string(notification.Type),
	//     },
	// })
	// if err != nil {
	//     return fmt.Errorf("ошибка отправки push уведомления: %w", err)
	// }

	return nil
}

// SendNewScheduleNotification отправляет уведомление о новом основном расписании
// В соответствии с ТЗ: "Новое основное расписание: ... Получатели: Все студенты и преподаватели"
func (s *Service) SendNewScheduleNotification(ctx context.Context, snapshot *schedule.ScheduleSnapshot) error {
	log.Println("Отправляем уведомление о новом основном расписании")

	title := "Обновлено расписание"
	message := fmt.Sprintf("Доступно новое расписание на период с %s по %s",
		snapshot.PeriodStart.Format("02.01.2006"),
		snapshot.PeriodEnd.Format("02.01.2006"))

	// TODO: Получить всех студентов и преподавателей
	// Пока используем заглушку
	var allUserIDs []uuid.UUID
	// allUserIDs = append(allUserIDs, studentIDs...)
	// allUserIDs = append(allUserIDs, teacherIDs...)

	// Если нет пользователей, выходим
	if len(allUserIDs) == 0 {
		log.Println("Нет пользователей для отправки уведомления о новом расписании")
		return nil
	}

	// Создаем уведомления для каждого пользователя
	var notificationErrors []error
	for _, userID := range allUserIDs {
		notification := &Notification{
			ID:          uuid.New(),
			UserID:      userID,
			Title:       title,
			Message:     message,
			Type:        NotificationTypeSystem,
			RelatedDate: snapshot.PeriodStart,
			IsRead:      false,
			CreatedAt:   time.Now(),
		}

		// Создаем уведомление в БД
		err := s.notificationRepo.CreateNotification(ctx, notification)
		if err != nil {
			notificationErrors = append(notificationErrors, fmt.Errorf("ошибка создания уведомления для пользователя %s: %w", userID, err))
			continue
		}

		log.Printf("Создано уведомление о новом расписании для пользователя %s", userID)

		// Отправляем push-уведомление
		if err := s.sendPushNotification(ctx, notification); err != nil {
			log.Printf("Ошибка отправки push уведомления пользователю %s: %v", userID, err)
		}
	}

	if len(notificationErrors) > 0 {
		// Возвращаем первую ошибку, если были ошибки
		return fmt.Errorf("ошибки при создании уведомлений: %v", notificationErrors[0])
	}

	log.Printf("Уведомление о новом расписании отправлено (%d пользователей)", len(allUserIDs))
	return nil
}

// MarkAsRead помечает уведомление как прочитанное
func (s *Service) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	return s.notificationRepo.MarkAsRead(ctx, notificationID)
}
