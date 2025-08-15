// Package gsheets предоставляет функции для работы с Google Таблицами через HTTP-запросы
// В соответствии с ТЗ: "Google Таблицы: ссылки на расписание и изменения"
// Реализация изменена для использования HTTP-запросов вместо Google Sheets API
// из-за невозможности настройки Google Cloud Billing.
package gsheets

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Client клиент для работы с Google Таблицами через HTTP-запросы
type Client struct {
	httpClient *http.Client
	// sheetGIDs - список gid листов для основного расписания.
	// Передается извне или задается по умолчанию.
	// Для таблицы изменений обычно используется gid=0 или он берется из конфига.
	sheetGIDs []int64
}

// NewClient создает новый клиент для Google Таблиц через HTTP-запросы.
// credentialsFile больше не используется, но сохранен для совместимости сигнатуры.
// sheetGIDs - список gid листов основного расписания.
func NewClient(sheetGIDs []int64) *Client {
	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Если список gid не передан, используем пустой
	if sheetGIDs == nil {
		sheetGIDs = []int64{}
	}

	return &Client{
		httpClient: client,
		sheetGIDs:  sheetGIDs,
	}
}

// ExportToCSVMainSchedule экспортирует основное расписание из Google Таблицы в CSV формат
// через HTTP-запросы для каждого листа (gid) и объединяет результаты.
// В соответствии с ТЗ: "Экспорт таблицы в CSV формат"
func (c *Client) ExportToCSVMainSchedule(ctx context.Context, sheetURL string) ([][]string, error) {
	log.Printf("Экспортируем основное расписание через HTTP-запросы: %s", sheetURL)

	// Извлекаем ID таблицы из URL
	spreadsheetID := c.extractSpreadsheetID(sheetURL)
	if spreadsheetID == "" {
		return nil, fmt.Errorf("не удалось извлечь ID таблицы из URL: %s", sheetURL)
	}

	log.Printf("Извлеченный ID таблицы: %s", spreadsheetID)

	// Проверяем, что список gid задан
	if len(c.sheetGIDs) == 0 {
		return nil, fmt.Errorf("список sheetGIDs пуст. Необходимо указать gid листов для основного расписания")
	}

	// Сбор данных со всех листов
	var allRecords [][]string

	for _, gid := range c.sheetGIDs {
		log.Printf("Экспортируем данные с листа gid=%d", gid)

		// Формируем URL для экспорта CSV конкретного листа
		exportURL := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv&gid=%d", spreadsheetID, gid)
		// ИСПРАВЛЕНО: Убраны лишние пробелы в начале URL

		// Создаем запрос с контекстом
		req, err := http.NewRequestWithContext(ctx, "GET", exportURL, nil)
		if err != nil {
			log.Printf("Ошибка создания запроса для gid=%d: %v", gid, err)
			continue // Продолжаем с другими листами
		}

		// Устанавливаем User-Agent, как в рабочем curl запросе
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

		// Выполняем запрос
		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("Ошибка выполнения запроса для gid=%d: %v", gid, err)
			continue
		}

		// Читаем тело ответа
		body, err := io.ReadAll(resp.Body)
		// Всегда закрываем тело, даже если была ошибка чтения
		resp.Body.Close()
		if err != nil {
			log.Printf("Ошибка чтения тела ответа для gid=%d: %v", gid, err)
			continue
		}

		// Проверяем статус ответа
		if resp.StatusCode != http.StatusOK {
			log.Printf("Неожиданный статус код для gid=%d: %d", gid, resp.StatusCode)
			continue
		}

		// Парсим CSV данные из тела ответа
		reader := csv.NewReader(bytes.NewReader(body))
		records, err := reader.ReadAll()
		if err != nil {
			log.Printf("Ошибка парсинга CSV для gid=%d: %v", gid, err)
			continue
		}

		log.Printf("Получено %d записей с листа gid=%d", len(records), gid)

		if len(allRecords) == 0 {
			// Первая порция данных - добавляем все, включая заголовок
			allRecords = append(allRecords, records...)
		} else {
			// Последующие порции - добавляем только данные (без заголовка)
			if len(records) > 1 {
				allRecords = append(allRecords, records[1:]...)
			}
		}
	}

	if len(allRecords) == 0 {
		log.Printf("Не удалось получить данные ни с одного листа таблицы: %s", spreadsheetID)
		return [][]string{}, nil // Возвращаем пустой массив, а не ошибку
	}

	log.Printf("Всего получено %d записей из всех листов таблицы: %s", len(allRecords), spreadsheetID)
	return allRecords, nil
}

// ExportToCSVChanges экспортирует изменения в расписании из Google Таблицы в CSV формат
// через HTTP-запрос. Использует gid=0 по умолчанию, если не указан другой.
func (c *Client) ExportToCSVChanges(ctx context.Context, sheetURL string, gid int64) ([][]string, error) {
	log.Printf("Экспортируем изменения через HTTP-запросы: %s, gid=%d", sheetURL, gid)

	// Извлекаем ID таблицы из URL
	spreadsheetID := c.extractSpreadsheetID(sheetURL)
	if spreadsheetID == "" {
		return nil, fmt.Errorf("не удалось извлечь ID таблицы из URL: %s", sheetURL)
	}

	log.Printf("Извеченный ID таблицы изменений: %s", spreadsheetID)

	// Формируем URL для экспорта CSV
	// ИСПРАВЛЕНО: Убраны лишние пробелы в начале URL
	exportURL := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv&gid=%d", spreadsheetID, gid)
	//              ^ Убраны пробелы здесь

	// Создаем запрос с контекстом
	req, err := http.NewRequestWithContext(ctx, "GET", exportURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Устанавливаем User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	// Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer func() {
		// Игнорируем ошибку закрытия тела ответа
		_ = resp.Body.Close()
	}()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("неожиданный статус код: %d", resp.StatusCode)
	}

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения тела ответа: %w", err)
	}

	// Парсим CSV данные из тела ответа
	reader := csv.NewReader(bytes.NewReader(body))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга CSV: %w", err)
	}

	log.Printf("Получено %d записей из таблицы изменений: %s", len(records), spreadsheetID)
	return records, nil
}

// ScheduleBellTimings представляет расписание звонков из фото
// Структура для хранения времени начала и окончания пар
type ScheduleBellTimings struct {
	DayOfWeek string
	Lessons   []LessonTiming
}

type LessonTiming struct {
	Number    int
	TimeStart string // "HH:MM"
	TimeEnd   string // "HH:MM"
}

// getBellTimings возвращает расписание звонков из фото
func getBellTimings() map[string][]LessonTiming {
	// Данные из фото
	// Преобразуем данные в более удобный для поиска формат
	timings := make(map[string][]LessonTiming)

	// Будние дни
	weekdayTimings := []LessonTiming{
		{1, "08:15", "09:00"},
		{2, "09:00", "09:45"},
		{3, "09:55", "10:40"},
		{4, "10:40", "11:25"},
		{5, "11:40", "12:25"},
		{6, "12:25", "13:10"},
		{7, "13:30", "14:15"},
		{8, "14:15", "15:00"},
		{9, "15:15", "16:00"},
		{10, "16:00", "16:45"},
		{11, "16:55", "17:40"},
		{12, "17:40", "18:25"},
	}

	// Суббота
	saturdayTimings := []LessonTiming{
		{1, "08:15", "09:00"},
		{2, "09:00", "09:45"},
		{3, "09:50", "10:35"},
		{4, "10:35", "11:20"},
		{5, "11:35", "12:20"},
		{6, "12:20", "13:05"},
		{7, "13:20", "14:05"},
		{8, "14:05", "14:50"},
		{9, "15:05", "15:50"},
		{10, "15:50", "16:35"},
		{11, "16:40", "17:25"},
		{12, "17:25", "18:10"},
	}

	// Заполняем карту для каждого дня недели
	weekdays := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница"}
	for _, day := range weekdays {
		timings[day] = weekdayTimings
	}
	timings["Суббота"] = saturdayTimings

	return timings
}

// ParseScheduleRecords парсит записи расписания из данных таблицы с горизонтальной структурой
// В соответствии с примером из ТЗ:
// Группа | Предмет | Преподаватель | Аудитория | Время начала | Время окончания | День недели
func (c *Client) ParseScheduleRecords(csvRecords [][]string) ([]ScheduleRecord, error) {
	if len(csvRecords) < 5 {
		return nil, fmt.Errorf("недостаточно данных в таблице (меньше 5 строк), получено: %d", len(csvRecords))
	}

	// --- Отладочное логирование ---
	log.Printf("DEBUG: Всего строк в CSV: %d", len(csvRecords))
	// Выведем первые несколько строк для анализа
	for i := 0; i < len(csvRecords) && i < 7; i++ {
		log.Printf("DEBUG: Строка CSV[%d]: %v", i, csvRecords[i])
	}
	// -----------------------------

	// Получаем расписание звонков из фото
	bellTimings := getBellTimings()

	// Извлекаем список групп из строки CSV[1]
	// Пример: [Группы - АТ 22-11, АТ 23-11, АТ 24-11, ДО 22-11-1, ДО 22-11-2          ]
	groupsLine := csvRecords[1]
	if len(groupsLine) == 0 {
		return nil, fmt.Errorf("строка с группами (CSV[1]) пуста")
	}
	// Первая ячейка содержит "Группы - ", остальные - названия групп
	var groupNames []string
	for i := 1; i < len(groupsLine); i++ {
		groupName := strings.TrimSpace(groupsLine[i])
		if groupName != "" {
			groupNames = append(groupNames, groupName)
		}
	}
	if len(groupNames) == 0 {
		return nil, fmt.Errorf("не удалось извлечь названия групп из строки CSV[1]")
	}
	log.Printf("DEBUG: Найденные группы: %v", groupNames)

	// Количество колонок данных на одну группу
	// Из строки заголовков CSV[4] видно, что на каждую группу приходится 4 колонки:
	// "Предмет", "вид занятия", "преподаватель", "Ауд."
	columnsPerGroup := 4

	// Проверяем, что структура данных соответствует ожиданиям
	headersLine := csvRecords[4]
	expectedHeaderColumns := len(groupNames) * columnsPerGroup
	if len(headersLine) < expectedHeaderColumns {
		return nil, fmt.Errorf("ожидалось как минимум %d колонок в строке заголовков (CSV[4]), но получено %d", expectedHeaderColumns, len(headersLine))
	}

	// Инициализируем список для результатов
	var records []ScheduleRecord

	// Переменная для хранения текущего дня недели и даты
	currentDayOfWeek := ""
	currentDateStr := ""
	// var currentDate time.Time // УДАЛЕНО: неиспользуемая переменная

	// Итерируемся по строкам с данными, начиная с CSV[5]
	for i := 5; i < len(csvRecords); i++ {
		row := csvRecords[i]

		// Пропускаем пустые строки
		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			continue
		}

		// Проверяем, является ли строка заголовком дня (содержит "День -")
		// Пример: [День - Понедельник, 23.06.2025          ]
		if len(row) > 0 && strings.Contains(strings.ToLower(row[0]), "день -") {
			// Извлекаем день недели и дату
			// row[0] = "День - Понедельник, 23.06.2025"
			parts := strings.Split(row[0], ",")
			if len(parts) >= 2 {
				// parts[0] = "День - Понедельник"
				dayPart := strings.TrimSpace(parts[0])
				dayParts := strings.Split(dayPart, "-")
				if len(dayParts) >= 2 {
					currentDayOfWeek = strings.TrimSpace(dayParts[1])
				}
				// parts[1] = " 23.06.2025"
				currentDateStr = strings.TrimSpace(parts[1])
				// УДАЛЕНО: currentDate, err := time.Parse("02.01.2006", currentDateStr)
				_, err := time.Parse("02.01.2006", currentDateStr) // ИЗМЕНЕНО: убрана переменная currentDate
				if err != nil {
					log.Printf("Предупреждение: Не удалось распарсить дату '%s' в строке %d: %v", currentDateStr, i, err)
					// УДАЛЕНО: currentDate = time.Time{} // Обнуляем дату в случае ошибки
				} else {
					log.Printf("DEBUG: Найден день: %s, дата: %s", currentDayOfWeek, currentDateStr)
				}
			}
			continue // Переходим к следующей строке
		}

		// Если это не строка с днем, считаем её строкой с данными
		// Проверяем, достаточно ли колонок
		if len(row) < expectedHeaderColumns {
			log.Printf("Предупреждение: Строка %d имеет недостаточно колонок (%d < %d), пропускаем", i, len(row), expectedHeaderColumns)
			continue
		}

		// Извлекаем номер пары из первой колонки
		lessonNumberStr := strings.TrimSpace(row[0])
		if lessonNumberStr == "" {
			log.Printf("Предупреждение: Пустой номер пары в строке %d, пропускаем", i)
			continue
		}
		lessonNumber, err := strconv.Atoi(lessonNumberStr)
		if err != nil {
			log.Printf("Предупреждение: Не удалось распарсить номер пары '%s' в строке %d: %v", lessonNumberStr, i, err)
			continue
		}

		// Получаем время начала и окончания для текущей пары и дня
		var timeStart, timeEnd string = "", ""
		if currentDayOfWeek != "" {
			// ИСПРАВЛЕНО: Используем правильный ключ для поиска расписания звонков
			if timingsForDay, ok := bellTimings[currentDayOfWeek]; ok {
				for _, timing := range timingsForDay {
					if timing.Number == lessonNumber {
						timeStart = timing.TimeStart
						timeEnd = timing.TimeEnd
						break
					}
				}
			}
			if timeStart == "" || timeEnd == "" {
				log.Printf("Предупреждение: Не найдено время для пары %d в день %s", lessonNumber, currentDayOfWeek)
			}
		} else {
			log.Printf("Предупреждение: Неизвестен день недели для строки %d, время не будет установлено", i)
		}

		// Итерируемся по группам и извлекаем данные
		for groupIndex, groupName := range groupNames {
			// Вычисляем начальный индекс колонок для текущей группы
			// Первая группа (индекс 0) -> колонки 1-4 (индексы 1,2,3,4)
			// Вторая группа (индекс 1) -> колонки 5-8 (индексы 5,6,7,8)
			// И т.д.
			startColIndex := 1 + (groupIndex * columnsPerGroup)
			endColIndex := startColIndex + columnsPerGroup - 1

			// Проверяем, что индексы валидны
			if endColIndex >= len(row) {
				log.Printf("Предупреждение: Индексы выходят за границы строки для группы %s в строке %d", groupName, i)
				continue
			}

			// Извлекаем данные для группы
			// row[startColIndex] = Предмет
			// row[startColIndex+1] = Вид занятия (игнорируем)
			// row[startColIndex+2] = Преподаватель
			// row[startColIndex+3] = Аудитория
			subject := strings.TrimSpace(row[startColIndex])
			// Вид занятия пропускаем
			teacher := strings.TrimSpace(row[startColIndex+2])
			classroom := strings.TrimSpace(row[startColIndex+3])

			// Пропускаем пустые записи
			if subject == "" {
				continue
			}

			// Создаем запись
			record := ScheduleRecord{
				GroupName: groupName,
				Subject:   subject,
				Teacher:   teacher,
				Classroom: classroom,
				TimeStart: timeStart,
				TimeEnd:   timeEnd,
				DayOfWeek: currentDayOfWeek,
			}

			// Добавляем дату, если она была распарсена
			// (Заполнение Date, PeriodStart, PeriodEnd будет происходить позже в другом месте,
			//  как это было в оригинальной логике)

			records = append(records, record)
		}
	}

	log.Printf("DEBUG: Успешно распаршено %d записей расписания", len(records))
	if len(records) > 0 {
		log.Printf("DEBUG: Пример первой записи: %+v", records[0])
	}
	return records, nil
}

// ParseChangeRecords парсит записи об изменениях из данных таблицы
// В соответствии с примером из ТЗ:
// Группа | Дата | Время начала | Время окончания | Предмет | Преподаватель | Аудитория | Тип изменения | Оригинальный предмет
func (c *Client) ParseChangeRecords(csvRecords [][]string) ([]ChangeRecord, error) {
	if len(csvRecords) < 2 {
		return nil, fmt.Errorf("недостаточно данных в таблице изменений (меньше 2 строк)")
	}

	// Находим индексы колонок в заголовке
	headers := csvRecords[0]
	var groupCol, dateCol, timeStartCol, timeEndCol, subjectCol, teacherCol, classroomCol, changeTypeCol, originalSubjectCol int = -1, -1, -1, -1, -1, -1, -1, -1, -1

	for i, header := range headers {
		headerStr := strings.TrimSpace(strings.ToLower(header))

		switch headerStr {
		case "группа":
			groupCol = i
		case "дата":
			dateCol = i
		case "время начала":
			timeStartCol = i
		case "время окончания":
			timeEndCol = i
		case "предмет":
			subjectCol = i
		case "преподаватель":
			teacherCol = i
		case "аудитория":
			classroomCol = i
		case "тип изменения":
			changeTypeCol = i
		case "оригинальный предмет":
			originalSubjectCol = i
		}
	}

	// Проверяем, что обязательные колонки найдены
	if groupCol == -1 || dateCol == -1 || timeStartCol == -1 || timeEndCol == -1 || subjectCol == -1 || changeTypeCol == -1 {
		return nil, fmt.Errorf("обязательные колонки для изменений не найдены в CSV заголовках. Найдено: группа=%d, дата=%d, время начала=%d, время окончания=%d, предмет=%d, тип изменения=%d",
			groupCol, dateCol, timeStartCol, timeEndCol, subjectCol, changeTypeCol)
	}

	var records []ChangeRecord

	// ИСПРАВЛЕНО: Используем индекс rowIndex для логирования
	for rowIndex, row := range csvRecords[1:] {
		if len(row) <= max(groupCol, dateCol, timeStartCol, timeEndCol, subjectCol, teacherCol, classroomCol, changeTypeCol, originalSubjectCol) {
			continue
		}

		dateStr := strings.TrimSpace(row[dateCol])
		// Ожидаемый формат даты из ТЗ: DD.MM.YYYY
		parsedDate, err := time.Parse("02.01.2006", dateStr)
		if err != nil {
			// Если не удалось распарсить дату, пропускаем строку
			// ИСПРАВЛЕНО: Добавлен индекс строки в лог (rowIndex+2, так как заголовок + сдвиг индекса)
			log.Printf("Ошибка парсинга даты '%s' в строке %d: %v", dateStr, rowIndex+2, err)
			continue
		}

		changeTypeStr := strings.TrimSpace(strings.ToLower(row[changeTypeCol]))
		var changeType string
		switch changeTypeStr {
		case "замена":
			changeType = "replacement"
		case "отмена":
			changeType = "cancellation"
		case "добавление":
			changeType = "addition"
		default:
			// Пропускаем строки с неизвестным типом изменения
			// ИСПРАВЛЕНО: Добавлен индекс строки в лог
			log.Printf("Неизвестный тип изменения '%s' в строке %d", changeTypeStr, rowIndex+2)
			continue
		}

		record := ChangeRecord{
			GroupName:       strings.TrimSpace(row[groupCol]),
			Date:            parsedDate,
			TimeStart:       strings.TrimSpace(row[timeStartCol]),
			TimeEnd:         strings.TrimSpace(row[timeEndCol]),
			Subject:         strings.TrimSpace(row[subjectCol]),
			Teacher:         strings.TrimSpace(row[teacherCol]),
			Classroom:       strings.TrimSpace(row[classroomCol]),
			ChangeType:      changeType,
			OriginalSubject: "", // По умолчанию пусто
		}

		// Если есть колонка "Оригинальный предмет", заполняем её
		if originalSubjectCol != -1 && originalSubjectCol < len(row) {
			record.OriginalSubject = strings.TrimSpace(row[originalSubjectCol])
		}

		// Базовая валидация
		if record.GroupName == "" || record.Subject == "" {
			// ИСПРАВЛЕНО: Добавлен индекс строки в лог
			log.Printf("Пропущена запись с пустой группой или предметом в строке %d", rowIndex+2)
			continue
		}

		// Валидация времени (если указаны)
		if record.TimeStart != "" {
			if _, err := time.Parse("15:04", record.TimeStart); err != nil {
				// ИСПРАВЛЕНО: Добавлен индекс строки в лог
				log.Printf("Некорректное время начала '%s' в строке %d: %v", record.TimeStart, rowIndex+2, err)
				// Не пропускаем, так как время может быть опциональным для некоторых типов изменений
			}
		}
		if record.TimeEnd != "" {
			if _, err := time.Parse("15:04", record.TimeEnd); err != nil {
				// ИСПРАВЛЕНО: Добавлен индекс строки в лог
				log.Printf("Некорректное время окончания '%s' в строке %d: %v", record.TimeEnd, rowIndex+2, err)
				// Не пропускаем, так как время может быть опциональным для некоторых типов изменений
			}
		}

		records = append(records, record)
	}

	return records, nil
}

// extractSpreadsheetID извлекает ID таблицы из URL
func (c *Client) extractSpreadsheetID(sheetURL string) string {
	// Пример URL:     https://docs.google.com/spreadsheets/d/ID/edit?usp=sharing
	// Извлекаем ID из URL

	// Убираем параметры из URL если есть
	if idx := strings.Index(sheetURL, "?"); idx != -1 {
		sheetURL = sheetURL[:idx]
	}
	if idx := strings.Index(sheetURL, "#"); idx != -1 {
		sheetURL = sheetURL[:idx]
	}

	// Извлекаем ID из пути
	// Пример:       https://docs.google.com/spreadsheets/d/ID/edit
	re := regexp.MustCompile(`/spreadsheets/d/([^/]+)`)
	matches := re.FindStringSubmatch(sheetURL)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// ScheduleRecord представляет запись из таблицы расписания
type ScheduleRecord struct {
	GroupName string `json:"group_name"`
	Subject   string `json:"subject"`
	Teacher   string `json:"teacher"`
	Classroom string `json:"classroom"`
	TimeStart string `json:"time_start"`
	TimeEnd   string `json:"time_end"`
	DayOfWeek string `json:"day_of_week"`
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
	ChangeType      string    `json:"change_type"` // "replacement", "cancellation", "addition"
	OriginalSubject string    `json:"original_subject"`
}

// max вспомогательная функция для нахождения максимума из списка int
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
