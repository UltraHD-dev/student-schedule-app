// Package gsheet предоставляет функции для работы с Google Таблицами
// В соответствии с ТЗ: "Google Таблицы: ссылки на расписание и изменения"
package gsheet

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
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
	exportURL := c.convertToExportURL(sheetURL)

	log.Printf("Экспортируем таблицу из %s (исходный URL: %s)", exportURL, sheetURL)

	// Выполняем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, "GET", exportURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	// Добавляем User-Agent для имитации браузера (иногда помогает)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		// Читаем тело ответа для отладки
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Таблица вернула статус %d. Тело ответа: %s", resp.StatusCode, string(body))

		// Если статус 404, логируем URL для отладки
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("Таблица не найдена: %s", exportURL)
		}
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
	// Логируем исходный URL для отладки
	log.Printf("Исходный URL таблицы: %s", sheetURL)

	// Убираем все параметры из URL если есть
	if idx := strings.Index(sheetURL, "?"); idx != -1 {
		sheetURL = sheetURL[:idx]
	}

	// Заменяем /edit на /export
	if strings.HasSuffix(sheetURL, "/edit") {
		sheetURL = sheetURL[:len(sheetURL)-5] // Убираем "/edit"
		sheetURL += "/export"
	} else if strings.Contains(sheetURL, "/d/") {
		// Если URL содержит ID таблицы, но не заканчивается на /edit
		// Добавляем /export
		if !strings.HasSuffix(sheetURL, "/") {
			sheetURL += "/"
		}
		sheetURL += "export"
	}

	// Добавляем параметры для экспорта в CSV
	exportURL := sheetURL + "?format=csv&gid=0"
	log.Printf("Сформированный URL для экспорта: %s", exportURL)

	return exportURL
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
