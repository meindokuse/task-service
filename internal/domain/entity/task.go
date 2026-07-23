package entity

import (
	"time"

	"github.com/meindokuse/task-service/internal/domain/valueobject"
)

type Task struct {
	ID          uint64
	TeamID      uint64
	Title       string
	Description string
	Status      valueobject.TaskStatus
	AssigneeID  *uint64
	CreatedBy   uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TaskFilter описывает параметры фильтрации и пагинации при получении списка задач.
type TaskFilter struct {
	TeamID     uint64
	Status     *valueobject.TaskStatus
	AssigneeID *uint64
	Page       int
	PageSize   int
}

type TaskList struct {
	Items    []Task
	Total    int
	Page     int
	PageSize int
}

type TaskHistory struct {
	ID        uint64
	TaskID    uint64
	ChangedBy uint64
	Field     string
	OldValue  string
	NewValue  string
	ChangedAt time.Time
}

type TaskComment struct {
	ID        uint64
	TaskID    uint64
	UserID    uint64
	Content   string
	CreatedAt time.Time
}
