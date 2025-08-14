// Package schedule предоставляет доступ к хранению расписания
// В соответствии с ТЗ: "Schedule Processing Service - обработка и хранение расписания"
package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository предоставляет доступ к хранению расписания
type Repository struct {
	db *sql.DB
}

// NewRepository создает новый репозиторий расписания
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateSnapshot создает новый снапшот расписания
func (r *Repository) CreateSnapshot(ctx context.Context, snapshot *ScheduleSnapshot) error {
	query := `
		INSERT INTO schedule_snapshots 
		(id, name, period_start, period_end, data, source_url, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`

	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		snapshot.ID,
		snapshot.Name,
		snapshot.PeriodStart,
		snapshot.PeriodEnd,
		snapshot.Data,
		snapshot.SourceURL,
		snapshot.IsActive).
		Scan(&createdAt)

	if err != nil {
		return fmt.Errorf("failed to create schedule snapshot: %w", err)
	}

	snapshot.CreatedAt = createdAt
	return nil
}

// GetActiveSnapshot получает активный снапшот расписания
func (r *Repository) GetActiveSnapshot(ctx context.Context) (*ScheduleSnapshot, error) {
	query := `
		SELECT id, name, period_start, period_end, data, created_at, source_url, is_active
		FROM schedule_snapshots
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT 1`

	snapshot := &ScheduleSnapshot{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&snapshot.ID,
		&snapshot.Name,
		&snapshot.PeriodStart,
		&snapshot.PeriodEnd,
		&snapshot.Data,
		&snapshot.CreatedAt,
		&snapshot.SourceURL,
		&snapshot.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active schedule snapshot found")
		}
		return nil, fmt.Errorf("failed to get active schedule snapshot: %w", err)
	}

	return snapshot, nil
}

// CreateChange создает новое изменение в расписании
func (r *Repository) CreateChange(ctx context.Context, change *ScheduleChange) error {
	query := `
		INSERT INTO schedule_changes 
		(id, snapshot_id, group_name, date, time_start, time_end, subject, teacher, classroom, change_type, original_subject, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at`

	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		change.ID,
		change.SnapshotID,
		change.GroupName,
		change.Date,
		change.TimeStart,
		change.TimeEnd,
		change.Subject,
		change.Teacher,
		change.Classroom,
		change.ChangeType,
		change.OriginalSubject,
		change.IsActive).
		Scan(&createdAt)

	if err != nil {
		return fmt.Errorf("failed to create schedule change: %w", err)
	}

	change.CreatedAt = createdAt
	return nil
}

// GetCurrentScheduleForGroup получает актуальное расписание для группы на определенную дату
func (r *Repository) GetCurrentScheduleForGroup(ctx context.Context, groupName string, date time.Time) ([]CurrentSchedule, error) {
	query := `
		SELECT id, group_name, date, time_start, time_end, subject, teacher, classroom, source_type, source_id, is_active
		FROM current_schedule
		WHERE group_name = $1 AND date = $2 AND is_active = true
		ORDER BY time_start`

	rows, err := r.db.QueryContext(ctx, query, groupName, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get current schedule for group: %w", err)
	}
	defer rows.Close()

	var schedules []CurrentSchedule
	for rows.Next() {
		var schedule CurrentSchedule
		err := rows.Scan(
			&schedule.ID,
			&schedule.GroupName,
			&schedule.Date,
			&schedule.TimeStart,
			&schedule.TimeEnd,
			&schedule.Subject,
			&schedule.Teacher,
			&schedule.Classroom,
			&schedule.SourceType,
			&schedule.SourceID,
			&schedule.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan current schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return schedules, nil
}

// BeginTx начинает транзакцию
func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

// GetCurrentScheduleEntry получает запись из current_schedule по группе, дате и времени начала
func (r *Repository) GetCurrentScheduleEntry(ctx context.Context, groupName string, date time.Time, timeStart string) (*CurrentSchedule, error) {
	query := `
		SELECT id, group_name, date, time_start, time_end, subject, teacher, classroom, source_type, source_id, is_active
		FROM current_schedule
		WHERE group_name = $1 AND date = $2 AND time_start = $3 AND is_active = true`

	entry := &CurrentSchedule{}
	err := r.db.QueryRowContext(ctx, query, groupName, date, timeStart).Scan(
		&entry.ID,
		&entry.GroupName,
		&entry.Date,
		&entry.TimeStart,
		&entry.TimeEnd,
		&entry.Subject,
		&entry.Teacher,
		&entry.Classroom,
		&entry.SourceType,
		&entry.SourceID,
		&entry.IsActive,
	)

	if err != nil {
		return nil, err
	}

	return entry, nil
}

// UpdateCurrentScheduleEntry обновляет запись в current_schedule
func (r *Repository) UpdateCurrentScheduleEntry(ctx context.Context, entry *CurrentSchedule) error {
	query := `
		UPDATE current_schedule
		SET subject = $1, teacher = $2, classroom = $3, source_type = $4, source_id = $5, is_active = $6
		WHERE id = $7`

	_, err := r.db.ExecContext(ctx, query,
		entry.Subject,
		entry.Teacher,
		entry.Classroom,
		entry.SourceType,
		entry.SourceID,
		entry.IsActive,
		entry.ID,
	)

	return err
}

// CreateCurrentScheduleEntry создает новую запись в current_schedule
func (r *Repository) CreateCurrentScheduleEntry(ctx context.Context, entry *CurrentSchedule) error {
	query := `
		INSERT INTO current_schedule 
		(id, group_name, date, time_start, time_end, subject, teacher, classroom, source_type, source_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID,
		entry.GroupName,
		entry.Date,
		entry.TimeStart,
		entry.TimeEnd,
		entry.Subject,
		entry.Teacher,
		entry.Classroom,
		entry.SourceType,
		entry.SourceID,
		entry.IsActive,
	)

	return err
}

// GetChangesForGroup получает изменения для группы на определенную дату
func (r *Repository) GetChangesForGroup(ctx context.Context, groupName string, date time.Time) ([]ScheduleChange, error) {
	query := `
		SELECT id, snapshot_id, group_name, date, time_start, time_end, subject, teacher, classroom, change_type, original_subject, created_at, is_active
		FROM schedule_changes
		WHERE group_name = $1 AND date = $2 AND is_active = true
		ORDER BY time_start`

	rows, err := r.db.QueryContext(ctx, query, groupName, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes for group: %w", err)
	}
	defer rows.Close()

	var changes []ScheduleChange
	for rows.Next() {
		var change ScheduleChange
		err := rows.Scan(
			&change.ID,
			&change.SnapshotID,
			&change.GroupName,
			&change.Date,
			&change.TimeStart,
			&change.TimeEnd,
			&change.Subject,
			&change.Teacher,
			&change.Classroom,
			&change.ChangeType,
			&change.OriginalSubject,
			&change.CreatedAt,
			&change.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan change: %w", err)
		}
		changes = append(changes, change)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return changes, nil
}
