// Package task_comment реализует интерфейс storage.Repository для комментариев к задачам.
package task_comment

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, c entity.TaskComment) (uint64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`,
		c.TaskID, c.UserID, c.Content,
	)
	if err != nil {
		return 0, terror.Internal("create task comment", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, terror.Internal("read last insert id", err)
	}
	return uint64(id), nil
}

func (r *Repository) ListByTask(ctx context.Context, taskID uint64) ([]entity.TaskComment, error) {
	var daos []taskCommentDAO
	err := r.db.SelectContext(ctx, &daos,
		`SELECT id, task_id, user_id, content, created_at FROM task_comments WHERE task_id = ? ORDER BY created_at ASC, id ASC`,
		taskID,
	)
	if err != nil {
		return nil, terror.Internal("list task comments", err)
	}

	items := make([]entity.TaskComment, 0, len(daos))
	for _, d := range daos {
		items = append(items, d.toEntity())
	}
	return items, nil
}
