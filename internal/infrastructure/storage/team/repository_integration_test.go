//go:build integration

package team_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/task"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team_member"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/user"
	"github.com/meindokuse/task-service/internal/testutil/mysqlcontainer"
)

func TestRepository_CreateGetByIDListForUser(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	users := user.New(db)
	teams := team.New(db)
	ctx := context.Background()

	ownerID, err := users.Create(ctx, entity.User{Email: "owner@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)

	teamID, err := teams.Create(ctx, entity.Team{Name: "Backend", CreatedBy: ownerID})
	require.NoError(t, err)

	byID, err := teams.GetByID(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, "Backend", byID.Name)

	members := team_member.New(db)
	require.NoError(t, members.Add(ctx, teamID, ownerID, valueobject.RoleOwner))

	list, err := teams.ListForUser(ctx, ownerID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, teamID, list[0].ID)
}

// TestRepository_Stats проверяет сложный запрос (a): для каждой команды —
// количество участников + количество завершённых за 7 дней задач, через
// JOIN 3 таблиц + агрегацию.
func TestRepository_Stats(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	users := user.New(db)
	teams := team.New(db)
	members := team_member.New(db)
	tasks := task.New(db)
	ctx := context.Background()

	ownerID, err := users.Create(ctx, entity.User{Email: "owner2@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	memberID, err := users.Create(ctx, entity.User{Email: "member2@example.com", PasswordHash: "h", Name: "Member"})
	require.NoError(t, err)

	teamID, err := teams.Create(ctx, entity.Team{Name: "Stats Team", CreatedBy: ownerID})
	require.NoError(t, err)
	require.NoError(t, members.Add(ctx, teamID, ownerID, valueobject.RoleOwner))
	require.NoError(t, members.Add(ctx, teamID, memberID, valueobject.RoleMember))

	doneID, err := tasks.Create(ctx, entity.Task{
		TeamID: teamID, Title: "Recently done", Status: valueobject.TaskStatusTodo, CreatedBy: ownerID,
	})
	require.NoError(t, err)
	require.NoError(t, tasks.Update(ctx, entity.Task{ID: doneID, TeamID: teamID, Title: "Recently done", Status: valueobject.TaskStatusDone, CreatedBy: ownerID}))

	_, err = tasks.Create(ctx, entity.Task{
		TeamID: teamID, Title: "Still open", Status: valueobject.TaskStatusTodo, CreatedBy: ownerID,
	})
	require.NoError(t, err)

	stats, err := teams.Stats(ctx)
	require.NoError(t, err)

	var found *entity.TeamStats
	for i := range stats {
		if stats[i].TeamID == teamID {
			found = &stats[i]
		}
	}
	require.NotNil(t, found, "stats for the created team must be present")
	assert.Equal(t, "Stats Team", found.Name)
	assert.Equal(t, 2, found.MemberCount)
	assert.Equal(t, 1, found.DoneLast7Days)
}
