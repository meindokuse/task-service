//go:build integration

package update_task_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/task/update_task"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/task"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/task_history"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team_member"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/user"
	"github.com/meindokuse/task-service/internal/pkg/txmanager"
	"github.com/meindokuse/task-service/internal/testutil/mysqlcontainer"
)

type noopCache struct{}

func (noopCache) InvalidateTeam(context.Context, uint64) error { return nil }

// TestService_Execute_ConcurrentUpdatesDoNotLoseWrites drives two concurrent
// Execute calls at the update_task service against a real MySQL container,
// proving the FOR UPDATE lock (added to fix the lost-update race) serializes
// them: both changes land, and the second one's history entry chains from
// the first's committed value rather than clobbering it silently.
func TestService_Execute_ConcurrentUpdatesDoNotLoseWrites(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	users := user.New(db)
	teams := team.New(db)
	members := team_member.New(db)
	tasks := task.New(db)
	history := task_history.New(db)
	tx := txmanager.New(db)

	ctx := context.Background()
	ownerID, err := users.Create(ctx, entity.User{Email: "concurrent-owner@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := teams.Create(ctx, entity.Team{Name: "Concurrent Team", CreatedBy: ownerID})
	require.NoError(t, err)
	taskID, err := tasks.Create(ctx, entity.Task{
		TeamID: teamID, Title: "Original", Status: valueobject.TaskStatusTodo, CreatedBy: ownerID,
	})
	require.NoError(t, err)

	svc := update_task.New(tasks, members, history, noopCache{}, tx)

	titleA := "Title from A"
	titleB := "Title from B"

	var wg sync.WaitGroup
	wg.Add(2)
	errs := make([]error, 2)
	go func() {
		defer wg.Done()
		_, errs[0] = svc.Execute(ctx, update_task.Request{TaskID: taskID, CallerID: ownerID, Title: &titleA})
	}()
	go func() {
		defer wg.Done()
		_, errs[1] = svc.Execute(ctx, update_task.Request{TaskID: taskID, CallerID: ownerID, Title: &titleB})
	}()
	wg.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])

	entries, err := history.ListByTask(ctx, taskID)
	require.NoError(t, err)
	require.Len(t, entries, 2, "both concurrent updates must be recorded, not just the last writer")

	final, err := tasks.GetByID(ctx, taskID)
	require.NoError(t, err)
	assert.Contains(t, []string{titleA, titleB}, final.Title)

	var first, second entity.TaskHistory
	for _, e := range entries {
		if e.OldValue == "Original" {
			first = e
		} else {
			second = e
		}
	}
	require.NotEmpty(t, first.NewValue, "one history entry must record the original title as its old value")
	assert.Equal(t, first.NewValue, second.OldValue,
		"the second update must have read the first update's committed title, proving no lost update")
	assert.ElementsMatch(t, []string{first.NewValue, second.NewValue}, []string{titleA, titleB})
}
