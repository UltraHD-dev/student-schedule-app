// Package scraper реализует Web Scraper Service для парсинга данных с сайта колледжа
// В соответствии с ТЗ: "Web Scraper Service - парсинг данных с сайта колледжа"
package scraper

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/changes"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/notifications"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/schedule"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/scraper/gsheet"
	"github.com/google/uuid"
)

type Service struct {
	httpClient          *http.Client
	gsheetClient        *gsheet.Client
	scheduleRepo        *schedule.Repository
	notificationService *notifications.Service // Полное имя типа
	changeService       *changes.Service       // Полное имя типа
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
	// Создаем контекст с таймаутом для HTTP-запроса
	httpCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	log.Printf("Отправляем запрос к %s", s.baseURL)
	req, err := http.NewRequestWithContext(httpCtx, "GET", s.baseURL, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	resp, err := s.httpClient.Do(req)
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
	// Уточняем селектор, чтобы найти именно таблицы расписания
	var sheetLinks []struct {
		URL       string
		Text      string
		Timestamp time.Time
	}

	// Ищем ссылки, которые могут содержать "расписание" или "schedule"
	doc.Find("a[href*='docs.google.com/spreadsheets']").Each(func(i int, selection *goquery.Selection) {
		href, exists := selection.Attr("href")
		if exists {
			text := strings.TrimSpace(selection.Text())
			lowerText := strings.ToLower(text)

			// Проверяем, что это не таблица изменений
			if !(strings.Contains(lowerText, "изменени") || strings.Contains(lowerText, "замены") || strings.Contains(lowerText, "замена")) {
				// Пытаемся извлечь дату из текста ссылки для определения свежести
				// Пример: "Расписание с 16.06.2025 по 22.06.2025"
				dateRegex := regexp.MustCompile(`(\d{2}\.\d{2}\.\d{4})`)
				dates := dateRegex.FindAllString(text, -1)
				var timestamp time.Time
				if len(dates) > 0 {
					// Берем первую найденную дату как дату начала периода
					timestamp, _ = time.Parse("02.01.2006", dates[0])
				} else {
					// Если дату не нашли, используем текущее время как fallback
					timestamp = time.Now()
				}

				sheetLinks = append(sheetLinks, struct {
					URL       string
					Text      string
					Timestamp time.Time
				}{
					URL:       href,
					Text:      text,
					Timestamp: timestamp,
				})
			}
		}
	})

	if len(sheetLinks) == 0 {
		// Если не нашли специфические ссылки, ищем любые таблицы
		doc.Find("a[href*='docs.google.com/spreadsheets']").Each(func(i int, selection *goquery.Selection) {
			href, exists := selection.Attr("href")
			if exists {
				text := strings.TrimSpace(selection.Text())
				// Берем первую попавшуюся таблицу как запасной вариант
				if len(sheetLinks) == 0 {
					sheetLinks = append(sheetLinks, struct {
						URL       string
						Text      string
						Timestamp time.Time
					}{
						URL:       href,
						Text:      text,
						Timestamp: time.Now(),
					})
				}
			}
		})
	}

	if len(sheetLinks) == 0 {
		return fmt.Errorf("не найдено ссылок на Google Таблицы с расписанием")
	}

	log.Printf("Найдено %d ссылок на Google Таблицы с расписанием", len(sheetLinks))

	// 3. Выбираем самую свежую таблицу (по дате в названии)
	// Сортируем по времени по убыванию
	sort.Slice(sheetLinks, func(i, j int) bool {
		return sheetLinks[i].Timestamp.After(sheetLinks[j].Timestamp)
	})

	// Берем первую (самую свежую) таблицу
	sheetURL := sheetLinks[0].URL
	log.Printf("Выбрана таблица: %s (дата: %s)", sheetLinks[0].Text, sheetLinks[0].Timestamp.Format("02.01.2006"))

	// 4. Экспорт таблицы в CSV формат
	// Проверим, можем ли мы экспортировать таблицу
	// Для этого просто попробуем получить метаданные таблицы
	log.Println("Проверяем доступность таблицы...")

	// Простая проверка доступности таблицы
	testResp, err := s.httpClient.Get(sheetURL)
	if err != nil {
		return fmt.Errorf("таблица недоступна: %w", err)
	}
	testResp.Body.Close()

	if testResp.StatusCode != http.StatusOK {
		return fmt.Errorf("таблица вернула статус %d", testResp.StatusCode)
	}

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

	// Определяем период действия расписания
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

	// В конце метода ScrapeMainSchedule, после создания снапшота:
	err = s.scheduleRepo.CreateSnapshot(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("ошибка создания снапшота расписания: %w", err)
	}

	log.Printf("Создан новый снапшот расписания: %s", snapshot.ID)

	// Отправляем уведомление о новом расписании
	if err := s.notificationService.SendNewScheduleNotification(ctx, snapshot); err != nil {
		log.Printf("Ошибка отправки уведомления о новом расписании: %v", err)
	}

	log.Println("Парсинг основного расписания завершен успешно")
	return nil
}

// ScrapeScheduleChanges парсит изменения в расписании
// В соответствии с ТЗ: "Процесс парсинга изменений"
func (s *Service) ScrapeScheduleChanges(ctx context.Context) error {
	log.Println("Начинаем парсинг изменений в расписании")

	// 1. Запрос к сайту колледжа для поиска ссылки на таблицу изменений
	log.Printf("Отправляем запрос к %s для поиска таблицы изменений", s.baseURL)

	// Создаем контекст с таймаутом для HTTP-запроса
	httpCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, "GET", s.baseURL, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка запроса к сайту колледжа: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сайт колледжа вернул статус %d", resp.StatusCode)
	}

	// 2. Парсим HTML и ищем ссылку на таблицу "Изменения в расписании"
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка парсинга HTML: %w", err)
	}

	// Ищем ссылку на таблицу изменений
	// Ищем ссылку, содержащую ключевые слова "изменени" или "замены"
	var changesURL string
	doc.Find("a[href*='docs.google.com/spreadsheets']").Each(func(i int, selection *goquery.Selection) {
		href, exists := selection.Attr("href")
		if exists {
			text := strings.ToLower(selection.Text())
			// Проверяем, содержит ли текст ключевые слова
			if strings.Contains(text, "изменени") || strings.Contains(text, "замены") || strings.Contains(text, "замена") {
				changesURL = href
				log.Printf("Найдена ссылка на таблицу изменений: %s", href)
				return // Прерываем перебор после первой найденной ссылки
			}
		}
	})

	// Если не нашли специфическую ссылку, пробуем найти любую таблицу с ключевыми словами в названии
	if changesURL == "" {
		doc.Find("a[href*='docs.google.com/spreadsheets']").Each(func(i int, selection *goquery.Selection) {
			href, exists := selection.Attr("href")
			if exists {
				// Берем первую попавшуюся таблицу как запасной вариант
				if changesURL == "" {
					changesURL = href
					log.Printf("Используем первую найденную таблицу как таблицу изменений: %s", href)
				}
			}
		})
	}

	// Если так и не нашли ссылку на таблицу изменений, выходим
	if changesURL == "" {
		log.Println("Не найдено ссылки на таблицу изменений. Пропускаем парсинг.")
		return nil
	}

	log.Printf("Используем таблицу изменений: %s", changesURL)

	// 3. Экспорт таблицы изменений в CSV формат
	log.Println("Экспортируем таблицу изменений в CSV")
	csvRecords, err := s.gsheetClient.ExportToCSV(ctx, changesURL)
	if err != nil {
		// Если таблица изменений не найдена или недоступна, это не критично
		log.Printf("Предупреждение: не удалось экспортировать таблицу изменений: %v", err)
		return nil
	}

	// 4. Парсинг данных об изменениях
	log.Println("Парсим данные об изменениях")
	changeRecords, err := s.parseChangeRecords(csvRecords)
	if err != nil {
		return fmt.Errorf("ошибка парсинга данных изменений: %w", err)
	}

	log.Printf("Успешно распаршено %d записей изменений", len(changeRecords))

	// 5. Сравнение с предыдущей версией (по хэшу данных)
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

	// 6. Если есть изменения - парсинг новых данных
	// 7. Создание записей в schedule_changes
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

	// 8. Обновление current_schedule
	// Вызываем Change Detection Service для применения изменений
	if len(createdChanges) > 0 {
		if err := s.changeService.ApplyChanges(ctx, createdChanges); err != nil {
			log.Printf("Ошибка применения изменений: %v", err)
			// Не возвращаем ошибку, чтобы не прерывать отправку уведомлений
		} else {
			log.Println("Изменения успешно применены к актуальному расписанию")
		}
	}

	// 9. Отправка уведомлений
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
	// Колонки могут называться по-разному, поэтому ищем по ключевым словам
	groupCol := -1
	dateCol := -1
	timeStartCol := -1
	timeEndCol := -1
	subjectCol := -1
	teacherCol := -1
	classroomCol := -1
	changeTypeCol := -1
	originalSubjectCol := -1

	for colName, colIndex := range columns {
		lowerColName := strings.ToLower(colName)
		switch {
		case strings.Contains(lowerColName, "группа"):
			groupCol = colIndex
		case strings.Contains(lowerColName, "дата"):
			dateCol = colIndex
		case strings.Contains(lowerColName, "время") && strings.Contains(lowerColName, "начало"):
			timeStartCol = colIndex
		case strings.Contains(lowerColName, "время") && strings.Contains(lowerColName, "окончание"):
			timeEndCol = colIndex
		case strings.Contains(lowerColName, "предмет"):
			subjectCol = colIndex
		case strings.Contains(lowerColName, "преподаватель"):
			teacherCol = colIndex
		case strings.Contains(lowerColName, "аудитория"):
			classroomCol = colIndex
		case strings.Contains(lowerColName, "тип") && (strings.Contains(lowerColName, "изменени") || strings.Contains(lowerColName, "замена")):
			changeTypeCol = colIndex
		case strings.Contains(lowerColName, "оригинальный") && strings.Contains(lowerColName, "предмет"):
			originalSubjectCol = colIndex
		}
	}

	// Проверяем, что нашли хотя бы основные колонки
	if groupCol == -1 || dateCol == -1 || timeStartCol == -1 || subjectCol == -1 {
		return nil, fmt.Errorf("не найдены обязательные колонки: Группа, Дата, Время начала, Предмет")
	}

	// Парсим данные
	var records []ChangeRecord
	for i := 1; i < len(csvRecords); i++ {
		row := csvRecords[i]
		if len(row) <= max(groupCol, dateCol, timeStartCol, subjectCol) {
			// Пропускаем неполные строки
			continue
		}

		// Парсим дату
		dateStr := strings.TrimSpace(row[dateCol])
		var date time.Time
		var err error

		// Пробуем разные форматы даты
		for _, format := range []string{"02.01.2006", "2006-01-02", "01/02/2006"} {
			date, err = time.Parse(format, dateStr)
			if err == nil {
				break
			}
		}

		if err != nil {
			log.Printf("Ошибка парсинга даты %s: %v", dateStr, err)
			continue
		}

		// Определяем тип изменения по значению в ячейке или по умолчанию
		changeType := "replacement" // По умолчанию замена
		if changeTypeCol != -1 && changeTypeCol < len(row) {
			changeTypeValue := strings.TrimSpace(row[changeTypeCol])
			switch strings.ToLower(changeTypeValue) {
			case "отмена", "cancelled", "cancellation":
				changeType = "cancellation"
			case "добавление", "added", "addition":
				changeType = "addition"
			case "замена", "replaced", "replacement":
				changeType = "replacement"
			}
		}

		record := ChangeRecord{
			GroupName:       strings.TrimSpace(row[groupCol]),
			Date:            date,
			TimeStart:       strings.TrimSpace(row[timeStartCol]),
			TimeEnd:         "",
			Subject:         strings.TrimSpace(row[subjectCol]),
			Teacher:         "",
			Classroom:       "",
			ChangeType:      changeType,
			OriginalSubject: "",
		}

		// Заполняем необязательные поля, если они есть
		if timeEndCol != -1 && timeEndCol < len(row) {
			record.TimeEnd = strings.TrimSpace(row[timeEndCol])
		}
		if teacherCol != -1 && teacherCol < len(row) {
			record.Teacher = strings.TrimSpace(row[teacherCol])
		}
		if classroomCol != -1 && classroomCol < len(row) {
			record.Classroom = strings.TrimSpace(row[classroomCol])
		}
		if originalSubjectCol != -1 && originalSubjectCol < len(row) {
			record.OriginalSubject = strings.TrimSpace(row[originalSubjectCol])
		}

		// Пропускаем пустые записи
		if record.GroupName == "" || record.Subject == "" {
			continue
		}

		records = append(records, record)
	}

	return records, nil
}

// Вспомогательная функция для нахождения максимума из нескольких int
func max(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
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
		// Немедленный запуск для тестирования
		log.Println("Немедленный запуск парсинга основного расписания")
		if err := s.ScrapeMainSchedule(ctx); err != nil {
			log.Printf("Ошибка при немедленном парсинге основного расписания: %v", err)
		}

		// Создаем таймер для еженедельного запуска (суббота ночью)
		// Пока используем более частый интервал для тестирования
		// В production будет 168 часов (неделя)
		ticker := time.NewTicker(1 * time.Hour) // В production будет time.NewTicker(168 * time.Hour)
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
		// Немедленный запуск для тестирования
		log.Println("Немедленный запуск парсинга изменений в расписании")
		if err := s.ScrapeScheduleChanges(ctx); err != nil {
			log.Printf("Ошибка при немедленном парсинге изменений в расписании: %v", err)
		}

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
		Period: fmt.Sprintf("%s - %s", time.Now().Format("02.01.2006"), time.Now().Add(7*24*time.Hour).Format("02.01.2006")),
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
