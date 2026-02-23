package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskStatus string

const (
	TaskStatusScheduled TaskStatus = "scheduled"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusCompleted TaskStatus = "completed"
)

type TriggerType string

const (
	TriggerTypeOneOff TriggerType = "one-off"
	TriggerTypeCron   TriggerType = "cron"
)

type Trigger struct {
	Type     TriggerType `json:"type" validate:"required,oneof=one-off cron"`
	DateTime *time.Time  `json:"datetime,omitempty"` // For one-off tasks (UTC)
	Cron     *string     `json:"cron,omitempty"`     // For cron tasks
}

func (t *Trigger) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, t)
}

func (t Trigger) Value() (driver.Value, error) {
	return json.Marshal(t)
}

type Action struct {
	Method  string            `json:"method" validate:"required"`
	URL     string            `json:"url" validate:"required,url"`
	Headers map[string]string `json:"headers,omitempty"`
	Payload json.RawMessage   `json:"payload,omitempty"`
}

func (a *Action) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, a)
}

func (a Action) Value() (driver.Value, error) {
	return json.Marshal(a)
}

type Task struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name      string     `gorm:"not null" json:"name" validate:"required"`
	Trigger   Trigger    `gorm:"type:jsonb;not null" json:"trigger" validate:"required"`
	Action    Action     `gorm:"type:jsonb;not null" json:"action" validate:"required"`
	Status    TaskStatus `gorm:"type:varchar(20);not null;default:'scheduled'" json:"status"`
	NextRun   *time.Time `gorm:"index" json:"next_run,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	Results   []TaskResult `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"-"`
}

func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (Task) TableName() string {
	return "tasks"
}
