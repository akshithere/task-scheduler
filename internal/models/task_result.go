package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskResult struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TaskID          uuid.UUID       `gorm:"type:uuid;not null;index" json:"task_id" validate:"required"`
	RunAt           time.Time       `gorm:"not null;index" json:"run_at"`
	StatusCode      int             `gorm:"not null" json:"status_code"`
	Success         bool            `gorm:"not null" json:"success"`
	ResponseHeaders json.RawMessage `gorm:"type:jsonb" json:"response_headers,omitempty"`
	ResponseBody    string          `gorm:"type:text" json:"response_body,omitempty"`
	ErrorMessage    *string         `gorm:"type:text" json:"error_message,omitempty"`
	DurationMs      int64           `gorm:"not null" json:"duration_ms"`
	CreatedAt       time.Time       `gorm:"autoCreateTime" json:"created_at"`
	Task            *Task           `gorm:"foreignKey:TaskID" json:"task,omitempty"`
}

func (tr *TaskResult) BeforeCreate(tx *gorm.DB) error {
	if tr.ID == uuid.Nil {
		tr.ID = uuid.New()
	}
	return nil
}

func (TaskResult) TableName() string {
	return "task_results"
}

type ResponseHeadersMap map[string][]string

func (r *ResponseHeadersMap) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, r)
}

func (r ResponseHeadersMap) Value() (driver.Value, error) {
	return json.Marshal(r)
}
