//go:build integration

package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/task"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team_member"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/user"
	"github.com/meindokuse/task-service/internal/pkg/terror"
	"github.com/meindokuse/task-service/internal/pkg/txmanager"
	"github.com/meindokuse/task-service/internal/testutil/mysqlcontainer"
)

type fixture struct {
	db      *sqlx.DB
	users   *user.Repository
	teams   *team.Repository
	members *team_member.Repository
	tasks   *task.Repository
}

func newFixture(t *testing.T) fixture {
	db := mysqlcontainer.Setup(t)
	return fixture{
		db:      db,
		users:   user.New(db),
		teams:   team.New(db),
		members: team_member.New(db),
		tasks:   task.New(db),
	}
}

func TestRepository_CreateGetByIDUpdate(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	ownerID, err := f.users.Create(ctx, entity.User{Email: "owner10@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := f.teams.Create(ctx, entity.Team{Name: "Task Team", CreatedBy: ownerID})
	require.NoError(t, err)

	taskID, err := f.tasks.Create(ctx, entity.Task{
		TeamID: teamID, Title: "Write docs", Description: "desc", Status: valueobject.TaskStatusTodo, CreatedBy: ownerID,
	})
	require.NoError(t, err)

	got, err := f.tasks.GetByID(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "Write docs", got.Title)
	assert.Equal(t, valueobject.TaskStatusTodo, got.Status)
	assert.Nil(t, got.AssigneeID)

	assignee := ownerID
	got.Status = valueobject.TaskStatusInProgress
	got.AssigneeID = &assignee
	require.NoError(t, f.tasks.Update(ctx, *got))

	updated, err := f.tasks.GetByID(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, valueobject.TaskStatusInProgress, updated.Status)
	require.NotNil(t, updated.AssigneeID)
	assert.Equal(t, ownerID, *updated.AssigneeID)
}

func TestRepository_UpdateNotFound(t *testing.T) {
	f := newFixture(t)
	err := f.tasks.Update(context.Background(), entity.Task{ID: 999999, Title: "x", Status: valueobject.TaskStatusTodo})
	require.Error(t, err)
	assert.Equal(t, 404, terror.HTTPStatus(err))
}

func TestRepository_ListFilterAndPagination(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	ownerID, err := f.users.Create(ctx, entity.User{Email: "owner11@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := f.teams.Create(ctx, entity.Team{Name: "List Team", CreatedBy: ownerID})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		status := valueobject.TaskStatusTodo
		if i%2 == 0 {
			status = valueobject.TaskStatusDone
		}
		_, err := f.tasks.Create(ctx, entity.Task{
			TeamID: teamID, Title: "Task", Status: status, CreatedBy: ownerID,
		})
		require.NoError(t, err)
	}

	done := valueobject.TaskStatusDone
	list, err := f.tasks.List(ctx, entity.TaskFilter{TeamID: teamID, Status: &done, Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, list.Total)
	assert.Len(t, list.Items, 2)
	assert.Equal(t, 1, list.Page)
	assert.Equal(t, 2, list.PageSize)

	page2, err := f.tasks.List(ctx, entity.TaskFilter{TeamID: teamID, Status: &done, Page: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, page2.Items, 1)
}

// TestRepository_TopCreators проверяет сложный запрос (b): топ-3 автора
// задач за текущий месяц по команде, через CTE + оконную функцию ROW_NUMBER.
func TestRepository_TopCreators(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	teamOwnerID, err := f.users.Create(ctx, entity.User{Email: "owner12@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := f.teams.Create(ctx, entity.Team{Name: "Rank Team", CreatedBy: teamOwnerID})
	require.NoError(t, err)

	prolific, err := f.users.Create(ctx, entity.User{Email: "prolific@example.com", PasswordHash: "h", Name: "Prolific"})
	require.NoError(t, err)
	occasional, err := f.users.Create(ctx, entity.User{Email: "occasional@example.com", PasswordHash: "h", Name: "Occasional"})
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err := f.tasks.Create(ctx, entity.Task{TeamID: teamID, Title: "T", Status: valueobject.TaskStatusTodo, CreatedBy: prolific})
		require.NoError(t, err)
	}
	_, err = f.tasks.Create(ctx, entity.Task{TeamID: teamID, Title: "T", Status: valueobject.TaskStatusTodo, CreatedBy: occasional})
	require.NoError(t, err)

	top, err := f.tasks.TopCreators(ctx, teamID)
	require.NoError(t, err)
	require.NotEmpty(t, top)
	assert.Equal(t, prolific, top[0].UserID)
	assert.Equal(t, 1, top[0].RankInTeam)
	assert.Equal(t, 3, top[0].TasksCount)
}

// TestRepository_OrphanedAssignees проверяет сложный запрос (c): задачи, чей
// исполнитель не является участником команды задачи, через подзапрос NOT EXISTS.
func TestRepository_OrphanedAssignees(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	ownerID, err := f.users.Create(ctx, entity.User{Email: "owner13@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := f.teams.Create(ctx, entity.Team{Name: "Integrity Team", CreatedBy: ownerID})
	require.NoError(t, err)
	require.NoError(t, f.members.Add(ctx, teamID, ownerID, valueobject.RoleOwner))

	outsider, err := f.users.Create(ctx, entity.User{Email: "outsider@example.com", PasswordHash: "h", Name: "Outsider"})
	require.NoError(t, err)

	orphanID, err := f.tasks.Create(ctx, entity.Task{
		TeamID: teamID, Title: "Assigned to outsider", Status: valueobject.TaskStatusTodo,
		AssigneeID: &outsider, CreatedBy: ownerID,
	})
	require.NoError(t, err)

	_, err = f.tasks.Create(ctx, entity.Task{
		TeamID: teamID, Title: "Assigned to owner", Status: valueobject.TaskStatusTodo,
		AssigneeID: &ownerID, CreatedBy: ownerID,
	})
	require.NoError(t, err)

	orphans, err := f.tasks.OrphanedAssignees(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, orphans, 1)
	assert.Equal(t, orphanID, orphans[0].ID)
}

// TestRepository_GetByIDForUpdate_SerializesConcurrentReaders proves the fix
// for the lost-update race in update_task: a SELECT ... FOR UPDATE taken
// inside one transaction blocks a second transaction's FOR UPDATE read on the
// same row until the first commits or rolls back.
func TestRepository_GetByIDForUpdate_SerializesConcurrentReaders(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	manager := txmanager.New(f.db)

	ownerID, err := f.users.Create(ctx, entity.User{Email: "lock-owner@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := f.teams.Create(ctx, entity.Team{Name: "Lock Team", CreatedBy: ownerID})
	require.NoError(t, err)
	taskID, err := f.tasks.Create(ctx, entity.Task{
		TeamID: teamID, Title: "Contended task", Status: valueobject.TaskStatusTodo, CreatedBy: ownerID,
	})
	require.NoError(t, err)

	lockedCh := make(chan struct{})
	releaseCh := make(chan struct{})
	firstTxErrCh := make(chan error, 1)

	go func() {
		firstTxErrCh <- manager.WithinTx(ctx, func(ctx context.Context) error {
			if _, err := f.tasks.GetByIDForUpdate(ctx, taskID); err != nil {
				return err
			}
			close(lockedCh)
			<-releaseCh
			return nil
		})
	}()

	<-lockedCh
	time.AfterFunc(150*time.Millisecond, func() { close(releaseCh) })

	start := time.Now()
	err = manager.WithinTx(ctx, func(ctx context.Context) error {
		_, err := f.tasks.GetByIDForUpdate(ctx, taskID)
		return err
	})
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NoError(t, <-firstTxErrCh)
	assert.GreaterOrEqual(t, elapsed, 140*time.Millisecond,
		"second reader should have blocked on the row lock until the first transaction released it")
}
