// Package gsheetapi предоставляет функции для работы с Google Sheets API
// В соответствии с ТЗ: "Google Таблицы: ссылки на расписание и изменения"
package gsheetapi

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Client клиент для работы с Google Sheets API
type Client struct {
	service *sheets.Service
}

// NewClient создает новый клиент для Google Sheets API
// В соответствии с ТЗ: "Web Scraper Service - парсинг данных с сайта колледжа"
func NewClient(credentialsFile string) (*Client, error) {
	// Создаем клиент Google Sheets API с аутентификацией через файл учетных данных
	service, err := sheets.NewService(context.Background(), option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания клиента Google Sheets API: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

// NewClientWithToken создает новый клиент для Google Sheets API с использованием токена
func NewClientWithToken(token string) (*Client, error) {
	// Создаем клиент Google Sheets API с использованием токена
	service, err := sheets.NewService(context.Background(), option.WithTokenSource(nil)) // TODO: Реализовать создание TokenSource
	if err != nil {
		return nil, fmt.Errorf("ошибка создания клиента Google Sheets API с токеном: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

// ExportToCSV экспортирует Google Таблицу в CSV формат через Google Sheets API
// В соответствии с ТЗ: "Экспорт таблицы в CSV формат"
func (c *Client) ExportToCSV(ctx context.Context, spreadsheetURL string) ([][]string, error) {
	log.Printf("Экспортируем таблицу через Google Sheets API: %s", spreadsheetURL)

	// Извлекаем ID таблицы из URL
	spreadsheetID := c.extractSpreadsheetID(spreadsheetURL)
	if spreadsheetID == "" {
		return nil, fmt.Errorf("не удалось извлечь ID таблицы из URL: %s", spreadsheetURL)
	}

	log.Printf("Извлеченный ID таблицы: %s", spreadsheetID)

	// Получаем данные из первой вкладки таблицы (gid=0)
	// Используем диапазон "A:Z" для получения всех колонок
	resp, err := c.service.Spreadsheets.Values.Get(spreadsheetID, "A:Z").Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных из Google Таблицы: %w", err)
	}

	// Если данных нет, возвращаем пустой массив
	if resp.Values == nil {
		log.Printf("В Google Таблице %s нет данных", spreadsheetID)
		return [][]string{}, nil
	}

	// Преобразуем данные в формат [][]string
	var records [][]string
	for _, row := range resp.Values {
		var record []string
		for _, cell := range row {
			// Преобразуем значение ячейки в строку
			if cellStr, ok := cell.(string); ok {
				record = append(record, cellStr)
			} else {
				// Если значение не строка, преобразуем в строку
				record = append(record, fmt.Sprintf("%v", cell))
			}
		}
		records = append(records, record)
	}

	log.Printf("Получено %d записей из Google Таблицы: %s", len(records), spreadsheetID)
	return records, nil
}

// ParseScheduleRecords парсит записи расписания из данных таблицы
// В соответствии с примером из ТЗ:
// Группа | Предмет | Преподаватель | Аудитория | Время начала | Время окончания | День недели
func (c *Client) ParseScheduleRecords(csvRecords [][]string) ([]ScheduleRecord, error) {
	if len(csvRecords) < 2 {
		return nil, fmt.Errorf("недостаточно данных в таблице")
	}

	// Находим индексы колонок в заголовке
	header := csvRecords[0]
	columns := map[string]int{}
	for i, col := range header {
		columns[strings.TrimSpace(col)] = i
	}

	// Проверяем наличие обязательных колонок для расписания
	requiredColumns := []string{"Группа", "Предмет", "Преподаватель", "Аудитория", "Время начала", "Время окончания", "День недели"}
	for _, col := range requiredColumns {
		if _, exists := columns[col]; !exists {
			return nil, fmt.Errorf("отсутствует обязательная колонка: %s", col)
		}
	}

	// Парсим данные
	var records []ScheduleRecord
	for i := 1; i < len(csvRecords); i++ {
		row := csvRecords[i]
		if len(row) < len(header) {
			// Пропускаем неполные строки
			continue
		}

		record := ScheduleRecord{
			GroupName: strings.TrimSpace(row[columns["Группа"]]),
			Subject:   strings.TrimSpace(row[columns["Предмет"]]),
			Teacher:   strings.TrimSpace(row[columns["Преподаватель"]]),
			Classroom: strings.TrimSpace(row[columns["Аудитория"]]),
			TimeStart: strings.TrimSpace(row[columns["Время начала"]]),
			TimeEnd:   strings.TrimSpace(row[columns["Время окончания"]]),
			DayOfWeek: strings.TrimSpace(row[columns["День недели"]]),
		}

		// Пропускаем пустые записи
		if record.GroupName == "" || record.Subject == "" {
			continue
		}

		records = append(records, record)
	}

	return records, nil
}

// ParseChangeRecords парсит записи об изменениях из данных таблицы
// В соответствии с примером из ТЗ:
// Группа | Дата | Время начала | Время окончания | Предмет | Преподаватель | Аудитория | Тип изменения | Оригинальный предмет
func (c *Client) ParseChangeRecords(csvRecords [][]string) ([]ChangeRecord, error) {
	if len(csvRecords) < 2 {
		return nil, fmt.Errorf("недостаточно данных в таблице изменений")
	}

	// Находим индексы колонок в заголовке
	header := csvRecords[0]
	columns := map[string]int{}
	for i, col := range header {
		columns[strings.TrimSpace(col)] = i
	}

	// Проверяем наличие обязательных колонок для изменений
	requiredColumns := []string{"Группа", "Дата", "Время начала", "Время окончания", "Предмет", "Преподаватель", "Аудитория", "Тип изменения"}
	for _, col := range requiredColumns {
		if _, exists := columns[col]; !exists {
			return nil, fmt.Errorf("отсутствует обязательная колонка в таблице изменений: %s", col)
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
			OriginalSubject: "", // По умолчанию пусто
		}

		// Если есть колонка "Оригинальный предмет", заполняем её
		if idx, exists := columns["Оригинальный предмет"]; exists && idx < len(row) {
			record.OriginalSubject = strings.TrimSpace(row[idx])
		}

		// Пропускаем пустые записи
		if record.GroupName == "" || record.Subject == "" {
			continue
		}

		records = append(records, record)
	}

	return records, nil
}

// extractSpreadsheetID извлекает ID таблицы из URL
func (c *Client) extractSpreadsheetID(sheetURL string) string {
	// Пример URL: https://docs.google.com/spreadsheets/d/ID/edit?usp=sharing
	// Извлекаем ID из URL

	// Убираем параметры из URL если есть
	if idx := strings.Index(sheetURL, "?"); idx != -1 {
		sheetURL = sheetURL[:idx]
	}

	// Извлекаем ID из пути
	// Пример: https://docs.google.com/spreadsheets/d/ID/edit
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
	ChangeType      string    `json:"change_type"`
	OriginalSubject string    `json:"original_subject"`
}
