// Package scraper реализует Web Scraper Service для парсинга данных с сайта колледжа
// В соответствии с ТЗ: "Web Scraper Service - парсинг данных с сайта колледжа"
package scraper

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/changes"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/notifications"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/schedule"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/scraper/gsheet"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strings"
	"time"
)

// Service предоставляет функции для парсинга данных с сайта колледжа
type Service struct {
	httpClient          *http.Client
	gsheetClient        *gsheet.Client
	scheduleRepo        *schedule.Repository
	notificationService *notifications.Service // Добавляем Notification Service
	changeService       *changes.Service       // Добавляем Change Detection Service
	baseURL             string
	lastChangeHash      string // Хэш последних данных об изменениях
}

// Config конфигурация scraper сервиса
type Config struct {
	BaseURL string
	Timeout time.Duration
}

// NewService создает новый scraper сервис
func NewService(config Config, scheduleRepo *schedule.Repository,
	notificationService *notifications.Service, changeService *changes.Service) *Service {
	return &Service{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		gsheetClient:        gsheet.NewClient(),
		scheduleRepo:        scheduleRepo,
		notificationService: notificationService,
		changeService:       changeService,
		baseURL:             config.BaseURL,
	}
}

// ScrapeMainSchedule парсит основное расписание с сайта колледжа
// В соответствии с ТЗ: "Процесс парсинга основного расписания"
func (s *Service) ScrapeMainSchedule(ctx context.Context) error {
	log.Println("Начинаем парсинг основного расписания с сайта колледжа")

	// 1. Запрос к https://kcpt72.ru/schedule/
	log.Printf("Отправляем запрос к %s", s.baseURL)
	resp, err := s.httpClient.Get(s.baseURL)
	if err != nil {
		return fmt.Errorf("ошибка запроса к сайту колледжа: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сайт колледжа вернул статус %d", resp.StatusCode)
	}

	// 2. Парсим HTML и ищем ссылки на Google Таблицы
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка парсинга HTML: %w", err)
	}

	// Ищем все ссылки на Google Таблицы
	var sheetLinks []string
	doc.Find("a[href*='docs.google.com/spreadsheets']").Each(func(i int, selection *goquery.Selection) {
		href, exists := selection.Attr("href")
		if exists {
			sheetLinks = append(sheetLinks, href)
		}
	})

	if len(sheetLinks) == 0 {
		return fmt.Errorf("не найдено ссылок на Google Таблицы")
	}

	log.Printf("Найдено %d ссылок на Google Таблицы", len(sheetLinks))

	// 3. Выбираем самую свежую таблицу (пока берем первую)
	// В реальной реализации нужно анализировать названия таблиц и выбирать самую свежую
	sheetURL := sheetLinks[0]
	log.Printf("Выбрана таблица: %s", sheetURL)

	// 4. Экспорт таблицы в CSV формат
	log.Println("Экспортируем таблицу в CSV")
	csvRecords, err := s.gsheetClient.ExportToCSV(ctx, sheetURL)
	if err != nil {
		return fmt.Errorf("ошибка экспорта таблицы в CSV: %w", err)
	}

	log.Printf("Получено %d записей из таблицы", len(csvRecords))

	// 5. Парсинг данных о расписании
	log.Println("Парсим данные о расписании")
	scheduleRecords, err := s.gsheetClient.ParseScheduleRecords(csvRecords)
	if err != nil {
		return fmt.Errorf("ошибка парсинга данных расписания: %w", err)
	}

	log.Printf("Успешно распаршено %d записей расписания", len(scheduleRecords))

	// 6. Создание нового снапшота в БД
	log.Println("Создаем новый снапшот расписания")

	// Преобразуем данные в формат JSON для хранения в БД
	scheduleData := s.convertToScheduleData(scheduleRecords)
	jsonData, err := json.Marshal(scheduleData)
	if err != nil {
		return fmt.Errorf("ошибка сериализации данных расписания в JSON: %w", err)
	}

	// Определяем период действия расписания (пока заглушка)
	periodStart := time.Now()
	periodEnd := periodStart.Add(7 * 24 * time.Hour) // +1 неделя

	// Создаем снапшот
	snapshot := &schedule.ScheduleSnapshot{
		ID:          uuid.New(),
		Name:        fmt.Sprintf("Расписание с %s", periodStart.Format("02.01.2006")),
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Data:        jsonData,
		SourceURL:   sheetURL,
		IsActive:    true,
	}

	err = s.scheduleRepo.CreateSnapshot(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("ошибка создания снапшота расписания: %w", err)
	}

	log.Printf("Создан новый снапшот расписания: %s", snapshot.ID)
	log.Println("Парсинг основного расписания завершен успешно")
	return nil
}

// ScrapeScheduleChanges парсит изменения в расписании
// В соответствии с ТЗ: "Процесс парсинга изменений"
func (s *Service) ScrapeScheduleChanges(ctx context.Context) error {
	log.Println("Начинаем парсинг изменений в расписании")

	// 1. Запрос к таблице "Изменения в расписании"
	// В реальной реализации нужно найти URL таблицы изменений на сайте колледжа
	// Пока используем заглушку
	changesURL := "https://docs.google.com/spreadsheets/d/CHANGE_TRACKING_SHEET_ID/edit"

	// 2. Экспорт таблицы изменений в CSV формат
	log.Println("Экспортируем таблицу изменений в CSV")
	csvRecords, err := s.gsheetClient.ExportToCSV(ctx, changesURL)
	if err != nil {
		// Если таблица изменений не найдена или недоступна, это не критично
		log.Printf("Предупреждение: не удалось экспортировать таблицу изменений: %v", err)
		return nil
	}

	// 3. Парсинг данных об изменениях
	log.Println("Парсим данные об изменениях")
	changeRecords, err := s.parseChangeRecords(csvRecords)
	if err != nil {
		return fmt.Errorf("ошибка парсинга данных изменений: %w", err)
	}

	log.Printf("Успешно распаршено %d записей изменений", len(changeRecords))

	// 4. Сравнение с предыдущей версией (по хэшу данных)
	currentHash, err := s.calculateDataHash(changeRecords)
	if err != nil {
		return fmt.Errorf("ошибка вычисления хэша данных: %w", err)
	}

	// Если данные не изменились, выходим
	if currentHash == s.lastChangeHash {
		log.Println("Нет новых изменений в расписании")
		return nil
	}

	log.Println("Обнаружены новые изменения в расписании")
	s.lastChangeHash = currentHash

	// 5. Если есть изменения - парсинг новых данных
	// 6. Создание записей в schedule_changes
	var createdChanges []schedule.ScheduleChange
	for _, record := range changeRecords {
		change := &schedule.ScheduleChange{
			ID:              uuid.New(),
			GroupName:       record.GroupName,
			Date:            record.Date,
			TimeStart:       record.TimeStart,
			TimeEnd:         record.TimeEnd,
			Subject:         record.Subject,
			Teacher:         record.Teacher,
			Classroom:       record.Classroom,
			ChangeType:      record.ChangeType,
			OriginalSubject: record.OriginalSubject,
			IsActive:        true,
		}

		err := s.scheduleRepo.CreateChange(ctx, change)
		if err != nil {
			log.Printf("Ошибка создания записи об изменении: %v", err)
			continue
		}

		log.Printf("Создана запись об изменении: %s для группы %s", change.ID, change.GroupName)
		createdChanges = append(createdChanges, *change)
	}

	// 7. Обновление current_schedule
	// TODO: Реализовать обновление current_schedule на основе новых изменений

	// 8. Отправка уведомлений
	for _, change := range createdChanges {
		// Отправляем уведомление через Notification Service
		if err := s.notificationService.SendScheduleChangeNotification(ctx, &change); err != nil {
			log.Printf("Ошибка отправки уведомления об изменении: %v", err)
		}
	}

	log.Println("Парсинг изменений в расписании завершен успешно")
	return nil
}

// parseChangeRecords парсит записи об изменениях из CSV данных
func (s *Service) parseChangeRecords(csvRecords [][]string) ([]ChangeRecord, error) {
	if len(csvRecords) < 2 {
		return nil, fmt.Errorf("недостаточно данных в CSV")
	}

	// Находим индексы колонок в заголовке
	header := csvRecords[0]
	columns := map[string]int{}
	for i, col := range header {
		columns[strings.TrimSpace(col)] = i
	}

	// Проверяем наличие обязательных колонок для изменений
	requiredColumns := []string{"Группа", "Дата", "Время начала", "Время окончания", "Предмет", "Тип изменения"}
	for _, col := range requiredColumns {
		if _, exists := columns[col]; !exists {
			return nil, fmt.Errorf("отсутствует обязательная колонка: %s", col)
		}
	}

	// Парсим данные
	var records []ChangeRecord
	for i := 1; i < len(csvRecords); i++ {
		row := csvRecords[i]
		if len(row) < len(header) {
			// Пропускаем неполные строки
			continue
		}

		// Парсим дату
		dateStr := strings.TrimSpace(row[columns["Дата"]])
		date, err := time.Parse("02.01.2006", dateStr)
		if err != nil {
			log.Printf("Ошибка парсинга даты %s: %v", dateStr, err)
			continue
		}

		record := ChangeRecord{
			GroupName:       strings.TrimSpace(row[columns["Группа"]]),
			Date:            date,
			TimeStart:       strings.TrimSpace(row[columns["Время начала"]]),
			TimeEnd:         strings.TrimSpace(row[columns["Время окончания"]]),
			Subject:         strings.TrimSpace(row[columns["Предмет"]]),
			Teacher:         strings.TrimSpace(row[columns["Преподаватель"]]),
			Classroom:       strings.TrimSpace(row[columns["Аудитория"]]),
			ChangeType:      strings.TrimSpace(row[columns["Тип изменения"]]),
			OriginalSubject: strings.TrimSpace(row[columns["Оригинальный предмет"]]),
		}

		// Пропускаем пустые записи
		if record.GroupName == "" || record.Subject == "" {
			continue
		}

		records = append(records, record)
	}

	return records, nil
}

// ChangeRecord представляет запись об изменении в расписании
type ChangeRecord struct {
	GroupName       string    `json:"group_name"`
	Date            time.Time `json:"date"`
	TimeStart       string    `json:"time_start"`
	TimeEnd         string    `json:"time_end"`
	Subject         string    `json:"subject"`
	Teacher         string    `json:"teacher"`
	Classroom       string    `json:"classroom"`
	ChangeType      string    `json:"change_type"`
	OriginalSubject string    `json:"original_subject"`
}

// StartPeriodicScraping запускает периодический парсинг
// В соответствии с ТЗ: "Еженедельно (суббота ночью)" и "Каждые 10 минут"
func (s *Service) StartPeriodicScraping(ctx context.Context) {
	// Горутина для парсинга основного расписания (еженедельно)
	go func() {
		// Создаем таймер для еженедельного запуска (суббота ночью)
		// Пока используем более частый интервал для тестирования
		ticker := time.NewTicker(1 * time.Hour) // В production будет 168 часов (неделя)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Проверяем, что сегодня суббота
				if time.Now().Weekday() == time.Saturday {
					if err := s.ScrapeMainSchedule(ctx); err != nil {
						log.Printf("Ошибка при парсинге основного расписания: %v", err)
					}
				}
			case <-ctx.Done():
				log.Println("Остановка периодического парсинга основного расписания")
				return
			}
		}
	}()

	// Горутина для парсинга изменений (каждые 10 минут)
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.ScrapeScheduleChanges(ctx); err != nil {
					log.Printf("Ошибка при парсинге изменений в расписании: %v", err)
				}
			case <-ctx.Done():
				log.Println("Остановка периодического парсинга изменений")
				return
			}
		}
	}()

	log.Println("Периодический парсинг запущен")
}

// convertToScheduleData преобразует записи расписания в структуру данных для JSON
func (s *Service) convertToScheduleData(records []gsheet.ScheduleRecord) *schedule.ScheduleData {
	// Группируем записи по группам и дням недели
	groups := make(map[string]map[string][]gsheet.ScheduleRecord)

	for _, record := range records {
		if _, exists := groups[record.GroupName]; !exists {
			groups[record.GroupName] = make(map[string][]gsheet.ScheduleRecord)
		}

		if _, exists := groups[record.GroupName][record.DayOfWeek]; !exists {
			groups[record.GroupName][record.DayOfWeek] = []gsheet.ScheduleRecord{}
		}

		groups[record.GroupName][record.DayOfWeek] = append(groups[record.GroupName][record.DayOfWeek], record)
	}

	// Преобразуем в формат ScheduleData
	scheduleData := &schedule.ScheduleData{
		Period: "TODO: определить период",
		Groups: make(map[string][]schedule.DaySchedule),
	}

	for groupName, days := range groups {
		var daySchedules []schedule.DaySchedule

		for dayName, dayRecords := range days {
			var lessons []schedule.Lesson

			for _, record := range dayRecords {
				lesson := schedule.Lesson{
					GroupName: record.GroupName,
					Subject:   record.Subject,
					Teacher:   record.Teacher,
					Classroom: record.Classroom,
					TimeStart: record.TimeStart,
					TimeEnd:   record.TimeEnd,
					DayOfWeek: record.DayOfWeek,
				}
				lessons = append(lessons, lesson)
			}

			daySchedule := schedule.DaySchedule{
				Day:     dayName,
				Lessons: lessons,
			}
			daySchedules = append(daySchedules, daySchedule)
		}

		scheduleData.Groups[groupName] = daySchedules
	}

	return scheduleData
}

// calculateDataHash вычисляет хэш данных для сравнения изменений
func (s *Service) calculateDataHash(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := md5.Sum(jsonData)
	return hex.EncodeToString(hash[:]), nil
}
