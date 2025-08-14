// Package schedule определяет модели данных для работы с расписанием
// В соответствии с ТЗ: "Schedule Processing Service - обработка и хранение расписания"
package schedule

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ScheduleSnapshot представляет снапшот расписания
// Соответствует таблице schedule_snapshots из ТЗ
type ScheduleSnapshot struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	PeriodStart time.Time `db:"period_start"`
	PeriodEnd   time.Time `db:"period_end"`
	Data        []byte    `db:"data"` // JSON данные в байтах
	CreatedAt   time.Time `db:"created_at"`
	SourceURL   string    `db:"source_url"`
	IsActive    bool      `db:"is_active"`
}

// ScheduleChange представляет изменение в расписании
// Соответствует таблице schedule_changes из ТЗ
type ScheduleChange struct {
	ID              uuid.UUID  `db:"id"`
	SnapshotID      *uuid.UUID `db:"snapshot_id"` // Может быть NULL
	GroupName       string     `db:"group_name"`
	Date            time.Time  `db:"date"`
	TimeStart       string     `db:"time_start"`
	TimeEnd         string     `db:"time_end"`
	Subject         string     `db:"subject"`
	Teacher         string     `db:"teacher"`
	Classroom       string     `db:"classroom"`
	ChangeType      string     `db:"change_type"`
	OriginalSubject string     `db:"original_subject"`
	CreatedAt       time.Time  `db:"created_at"`
	IsActive        bool       `db:"is_active"`
}

// CurrentSchedule представляет актуальное расписание
// Соответствует таблице current_schedule из ТЗ
type CurrentSchedule struct {
	ID         uuid.UUID `db:"id"`
	GroupName  string    `db:"group_name"`
	Date       time.Time `db:"date"`
	TimeStart  string    `db:"time_start"`
	TimeEnd    string    `db:"time_end"`
	Subject    string    `db:"subject"`
	Teacher    string    `db:"teacher"`
	Classroom  string    `db:"classroom"`
	SourceType string    `db:"source_type"`
	SourceID   uuid.UUID `db:"source_id"`
	IsActive   bool      `db:"is_active"`
}

// Lesson представляет одну пару в расписании
// Используется при парсинге данных из таблиц
type Lesson struct {
	GroupName string `json:"group_name" csv:"Группа"`
	Subject   string `json:"subject" csv:"Предмет"`
	Teacher   string `json:"teacher" csv:"Преподаватель"`
	Classroom string `json:"classroom" csv:"Аудитория"`
	TimeStart string `json:"time_start" csv:"Время начала"`
	TimeEnd   string `json:"time_end" csv:"Время окончания"`
	DayOfWeek string `json:"day_of_week" csv:"День недели"`
}

// ScheduleData представляет структуру данных расписания для JSON
// В соответствии с примером из ТЗ
type ScheduleData struct {
	Period string                   `json:"period"`
	Groups map[string][]DaySchedule `json:"groups"`
}

// DaySchedule представляет расписание на один день
type DaySchedule struct {
	Day     string   `json:"day"`
	Lessons []Lesson `json:"lessons"`
}

// Value реализует интерфейс driver.Valuer для ScheduleData
func (sd ScheduleData) Value() (driver.Value, error) {
	return json.Marshal(sd)
}

// Scan реализует интерфейс sql.Scanner для ScheduleData
func (sd *ScheduleData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into ScheduleData", value)
	}

	return json.Unmarshal(bytes, sd)
}
