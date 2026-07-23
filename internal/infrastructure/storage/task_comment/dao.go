package task_comment

import (
	"time"

	"github.com/meindokuse/task-service/internal/domain/entity"
)

type taskCommentDAO struct {
	ID        uint64    `db:"id"`
	TaskID    uint64    `db:"task_id"`
	UserID    uint64    `db:"user_id"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
}

func (d taskCommentDAO) toEntity() entity.TaskComment {
	return entity.TaskComment{
		ID:        d.ID,
		TaskID:    d.TaskID,
		UserID:    d.UserID,
		Content:   d.Content,
		CreatedAt: d.CreatedAt,
	}
}
