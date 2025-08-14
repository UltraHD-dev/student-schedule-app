// Package gsheet предоставляет функции для работы с Google Таблицами
// В соответствии с ТЗ: "Google Таблицы: ссылки на расписание и изменения"
package gsheet

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client клиент для работы с Google Таблицами
type Client struct {
	httpClient *http.Client
}

// NewClient создает новый клиент для Google Таблиц
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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

// ExportToCSV экспортирует Google Таблицу в CSV формат
// В соответствии с ТЗ: "Экспорт таблицы в CSV формат"
func (c *Client) ExportToCSV(ctx context.Context, sheetURL string) ([][]string, error) {
	// Преобразуем URL Google Таблицы в URL для экспорта в CSV
	// Пример: https://docs.google.com/spreadsheets/d/ID/edit -> https://docs.google.com/spreadsheets/d/ID/export?format=csv
	exportURL := c.convertToExportURL(sheetURL)

	// Выполняем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, "GET", exportURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google Таблица вернула статус %d", resp.StatusCode)
	}

	// Читаем CSV данные
	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения CSV данных: %w", err)
	}

	return records, nil
}

// convertToExportURL преобразует URL Google Таблицы в URL для экспорта
func (c *Client) convertToExportURL(sheetURL string) string {
	// Убираем параметры из URL если есть
	if idx := strings.Index(sheetURL, "?"); idx != -1 {
		sheetURL = sheetURL[:idx]
	}

	// Добавляем параметры для экспорта в CSV
	if !strings.HasSuffix(sheetURL, "/") {
		sheetURL += "/"
	}

	return sheetURL + "export?format=csv"
}

// ParseScheduleRecords парсит записи расписания из CSV данных
// В соответствии с примером из ТЗ:
// Группа | Предмет | Преподаватель | Аудитория | Время начала | Время окончания | День недели
func (c *Client) ParseScheduleRecords(csvRecords [][]string) ([]ScheduleRecord, error) {
	if len(csvRecords) < 2 {
		return nil, fmt.Errorf("недостаточно данных в CSV")
	}

	// Находим индексы колонок в заголовке
	header := csvRecords[0]
	columns := map[string]int{}
	for i, col := range header {
		columns[strings.TrimSpace(col)] = i
	}

	// Проверяем наличие обязательных колонок
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
