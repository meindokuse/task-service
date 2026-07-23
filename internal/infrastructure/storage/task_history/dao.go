package task_history

import (
	"database/sql"
	"time"

	"github.com/meindokuse/task-service/internal/domain/entity"
)

type taskHistoryDAO struct {
	ID        uint64         `db:"id"`
	TaskID    uint64         `db:"task_id"`
	ChangedBy uint64         `db:"changed_by"`
	Field     string         `db:"field_name"`
	OldValue  sql.NullString `db:"old_value"`
	NewValue  sql.NullString `db:"new_value"`
	ChangedAt time.Time      `db:"changed_at"`
}

func (d taskHistoryDAO) toEntity() entity.TaskHistory {
	return entity.TaskHistory{
		ID:        d.ID,
		TaskID:    d.TaskID,
		ChangedBy: d.ChangedBy,
		Field:     d.Field,
		OldValue:  d.OldValue.String,
		NewValue:  d.NewValue.String,
		ChangedAt: d.ChangedAt,
	}
}
