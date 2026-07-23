//go:build integration

package task_history_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/task"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/task_history"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/user"
	"github.com/meindokuse/task-service/internal/pkg/txmanager"
	"github.com/meindokuse/task-service/internal/testutil/mysqlcontainer"
)

func TestRepository_CreateBatchAndListByTask(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	users := user.New(db)
	teams := team.New(db)
	tasks := task.New(db)
	history := task_history.New(db)
	ctx := context.Background()

	ownerID, err := users.Create(ctx, entity.User{Email: "owner20@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := teams.Create(ctx, entity.Team{Name: "History Team", CreatedBy: ownerID})
	require.NoError(t, err)
	taskID, err := tasks.Create(ctx, entity.Task{TeamID: teamID, Title: "T", Status: valueobject.TaskStatusTodo, CreatedBy: ownerID})
	require.NoError(t, err)

	err = history.CreateBatch(ctx, []entity.TaskHistory{
		{TaskID: taskID, ChangedBy: ownerID, Field: "title", OldValue: "old", NewValue: "new"},
		{TaskID: taskID, ChangedBy: ownerID, Field: "status", OldValue: "todo", NewValue: "done"},
	})
	require.NoError(t, err)

	entries, err := history.ListByTask(ctx, taskID)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	// ListByTask сортирует по убыванию даты (сначала новые).
	assert.Equal(t, "status", entries[0].Field)
	assert.Equal(t, "title", entries[1].Field)
}

// TestRepository_CreateBatch_RollsBackWithTx проверяет, что если CreateBatch
// выполняется внутри транзакции txmanager, которая затем завершается ошибкой,
// то ни одна из его строк не сохраняется — именно на этой атомарности
// строится use case update_task.
func TestRepository_CreateBatch_RollsBackWithTx(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	users := user.New(db)
	teams := team.New(db)
	tasks := task.New(db)
	history := task_history.New(db)
	tx := txmanager.New(db)
	ctx := context.Background()

	ownerID, err := users.Create(ctx, entity.User{Email: "owner21@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := teams.Create(ctx, entity.Team{Name: "Rollback Team", CreatedBy: ownerID})
	require.NoError(t, err)
	taskID, err := tasks.Create(ctx, entity.Task{TeamID: teamID, Title: "T", Status: valueobject.TaskStatusTodo, CreatedBy: ownerID})
	require.NoError(t, err)

	boom := errors.New("boom")
	err = tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := history.CreateBatch(ctx, []entity.TaskHistory{
			{TaskID: taskID, ChangedBy: ownerID, Field: "title", OldValue: "old", NewValue: "new"},
		}); err != nil {
			return err
		}
		return boom
	})
	require.ErrorIs(t, err, boom)

	entries, err := history.ListByTask(ctx, taskID)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
