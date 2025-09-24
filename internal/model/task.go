// Package model содержит модели данных приложения.
package model

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// TaskType представляет тип задачи
type TaskType string

const (
	TaskTypeParseReleases  TaskType = "parse_releases"
	TaskTypeUpdatePlaylist TaskType = "update_playlist"
	TaskTypeUpdateHomework TaskType = "update_homework"
	TaskTypeHomeworkReset  TaskType = "homework_reset"
)

// IsValid проверяет валидность типа задачи
func (t TaskType) IsValid() bool {
	switch t {
	case TaskTypeParseReleases, TaskTypeUpdatePlaylist, TaskTypeUpdateHomework, TaskTypeHomeworkReset:
		return true
	default:
		return false
	}
}

// String возвращает строковое представление типа задачи
func (t TaskType) String() string {
	return string(t)
}

// MarshalText реализует encoding.TextMarshaler
func (t TaskType) MarshalText() ([]byte, error) {
	return []byte(string(t)), nil
}

// UnmarshalText реализует encoding.TextUnmarshaler
func (t *TaskType) UnmarshalText(data []byte) error {
	*t = TaskType(data)
	return nil
}

// Task представляет задачу в системе
type Task struct {
	bun.BaseModel `bun:"table:tasks"`

	TaskID         int                    `bun:"task_id,pk,autoincrement" json:"task_id"`
	Name           string                 `bun:"name,unique,notnull" json:"name"`
	Description    string                 `bun:"description" json:"description"`
	TaskType       TaskType               `bun:"task_type,notnull" json:"task_type"`
	CronExpression string                 `bun:"cron_expression,notnull" json:"cron_expression"`
	IsActive       bool                   `bun:"is_active,notnull,default:true" json:"is_active"`
	LastRun        *time.Time             `bun:"last_run" json:"last_run"`
	NextRun        *time.Time             `bun:"next_run" json:"next_run"`
	RunCount       int                    `bun:"run_count,notnull,default:0" json:"run_count"`
	SuccessCount   int                    `bun:"success_count,notnull,default:0" json:"success_count"`
	ErrorCount     int                    `bun:"error_count,notnull,default:0" json:"error_count"`
	LastError      string                 `bun:"last_error" json:"last_error"`
	Config         map[string]interface{} `bun:"config,type:jsonb" json:"config"`
	CreatedAt      time.Time              `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time              `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// Validate проверяет валидность задачи
func (t *Task) Validate() error {
	var errors ValidationErrors

	if t.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: "name is required"})
	}

	if !t.TaskType.IsValid() {
		errors = append(errors, ValidationError{Field: "task_type", Message: "invalid task type"})
	}

	if t.CronExpression == "" {
		errors = append(errors, ValidationError{Field: "cron_expression", Message: "cron_expression is required"})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// IsValid проверяет валидность задачи
func (t *Task) IsValid() bool {
	return t.Name != "" && t.TaskType.IsValid() && t.CronExpression != ""
}

// GetConfigValue получает значение из конфигурации
func (t *Task) GetConfigValue(key string) (interface{}, bool) {
	if t.Config == nil {
		return nil, false
	}
	value, exists := t.Config[key]
	return value, exists
}

// SetConfigValue устанавливает значение в конфигурации
func (t *Task) SetConfigValue(key string, value interface{}) {
	if t.Config == nil {
		t.Config = make(map[string]interface{})
	}
	t.Config[key] = value
}

// GetConfigString получает строковое значение из конфигурации
func (t *Task) GetConfigString(key string) (string, bool) {
	value, exists := t.GetConfigValue(key)
	if !exists {
		return "", false
	}
	if str, ok := value.(string); ok {
		return str, true
	}
	return "", false
}

// GetConfigInt получает целочисленное значение из конфигурации
func (t *Task) GetConfigInt(key string) (int, bool) {
	value, exists := t.GetConfigValue(key)
	if !exists {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), true
		}
	}
	return 0, false
}

// GetConfigBool получает булево значение из конфигурации
func (t *Task) GetConfigBool(key string) (bool, bool) {
	value, exists := t.GetConfigValue(key)
	if !exists {
		return false, false
	}
	if b, ok := value.(bool); ok {
		return b, true
	}
	return false, false
}

// UpdateRunStats обновляет статистику выполнения задачи
func (t *Task) UpdateRunStats(success bool, err error) {
	t.RunCount++
	if success {
		t.SuccessCount++
		t.LastError = ""
	} else {
		t.ErrorCount++
		if err != nil {
			t.LastError = err.Error()
		}
	}
	now := time.Now()
	t.LastRun = &now
}

// TaskRepository определяет интерфейс для работы с задачами
type TaskRepository interface {
	Repository[Task]
	GetByType(taskType TaskType) ([]Task, error)
	GetActive() ([]Task, error)
	GetDueTasks() ([]Task, error)
	UpdateRunStats(taskID int, success bool, err error) error
}
