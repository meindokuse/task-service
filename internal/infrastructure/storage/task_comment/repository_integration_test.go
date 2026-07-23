//go:build integration

package task_comment_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/task"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/task_comment"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/user"
	"github.com/meindokuse/task-service/internal/testutil/mysqlcontainer"
)

func TestRepository_CreateAndListByTask(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	users := user.New(db)
	teams := team.New(db)
	tasks := task.New(db)
	comments := task_comment.New(db)
	ctx := context.Background()

	ownerID, err := users.Create(ctx, entity.User{Email: "owner30@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := teams.Create(ctx, entity.Team{Name: "Comment Team", CreatedBy: ownerID})
	require.NoError(t, err)
	taskID, err := tasks.Create(ctx, entity.Task{TeamID: teamID, Title: "T", Status: valueobject.TaskStatusTodo, CreatedBy: ownerID})
	require.NoError(t, err)

	id1, err := comments.Create(ctx, entity.TaskComment{TaskID: taskID, UserID: ownerID, Content: "first"})
	require.NoError(t, err)
	id2, err := comments.Create(ctx, entity.TaskComment{TaskID: taskID, UserID: ownerID, Content: "second"})
	require.NoError(t, err)

	list, err := comments.ListByTask(ctx, taskID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, id1, list[0].ID)
	assert.Equal(t, id2, list[1].ID)
	assert.Equal(t, "first", list[0].Content)
}
