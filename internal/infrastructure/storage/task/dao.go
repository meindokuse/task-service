package task

import (
	"database/sql"
	"time"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
)

type taskDAO struct {
	ID          uint64        `db:"id"`
	TeamID      uint64        `db:"team_id"`
	Title       string        `db:"title"`
	Description string        `db:"description"`
	Status      string        `db:"status"`
	AssigneeID  sql.NullInt64 `db:"assignee_id"`
	CreatedBy   uint64        `db:"created_by"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
}

func (d taskDAO) toEntity() entity.Task {
	t := entity.Task{
		ID:          d.ID,
		TeamID:      d.TeamID,
		Title:       d.Title,
		Description: d.Description,
		Status:      valueobject.TaskStatus(d.Status),
		CreatedBy:   d.CreatedBy,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	if d.AssigneeID.Valid {
		id := uint64(d.AssigneeID.Int64)
		t.AssigneeID = &id
	}
	return t
}
