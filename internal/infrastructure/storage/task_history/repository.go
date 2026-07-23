// Package task_history реализует интерфейс storage.Repository для истории
// изменений задач: при каждом обновлении задачи на каждое изменённое поле
// записывается отдельная строка.
package task_history

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
	"github.com/meindokuse/task-service/internal/pkg/txmanager"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// CreateBatch записывает несколько строк истории (по одной на изменённое поле).
// Если ctx несёт активную транзакцию (см. txmanager), запись выполняется в её
// рамках, чтобы обновление задачи и его история оставались атомарными.
func (r *Repository) CreateBatch(ctx context.Context, entries []entity.TaskHistory) error {
	if len(entries) == 0 {
		return nil
	}

	exec := txmanager.Ext(ctx, r.db)
	for _, e := range entries {
		_, err := sqlx.NamedExecContext(ctx, exec, `
			INSERT INTO task_history (task_id, changed_by, field_name, old_value, new_value)
			VALUES (:task_id, :changed_by, :field_name, :old_value, :new_value)`,
			map[string]interface{}{
				"task_id":    e.TaskID,
				"changed_by": e.ChangedBy,
				"field_name": e.Field,
				"old_value":  e.OldValue,
				"new_value":  e.NewValue,
			},
		)
		if err != nil {
			return terror.Internal("insert task history", err)
		}
	}
	return nil
}

func (r *Repository) ListByTask(ctx context.Context, taskID uint64) ([]entity.TaskHistory, error) {
	var daos []taskHistoryDAO
	err := r.db.SelectContext(ctx, &daos, `
		SELECT id, task_id, changed_by, field_name, old_value, new_value, changed_at
		FROM task_history WHERE task_id = ? ORDER BY changed_at DESC, id DESC`,
		taskID,
	)
	if err != nil {
		return nil, terror.Internal("list task history", err)
	}

	items := make([]entity.TaskHistory, 0, len(daos))
	for _, d := range daos {
		items = append(items, d.toEntity())
	}
	return items, nil
}
