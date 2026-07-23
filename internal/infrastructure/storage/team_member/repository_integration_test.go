//go:build integration

package team_member_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team_member"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/user"
	"github.com/meindokuse/task-service/internal/pkg/terror"
	"github.com/meindokuse/task-service/internal/testutil/mysqlcontainer"
)

func TestRepository_AddGetRoleIsMemberListMembers(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	users := user.New(db)
	teams := team.New(db)
	members := team_member.New(db)
	ctx := context.Background()

	ownerID, err := users.Create(ctx, entity.User{Email: "owner3@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	memberID, err := users.Create(ctx, entity.User{Email: "member3@example.com", PasswordHash: "h", Name: "Member"})
	require.NoError(t, err)

	teamID, err := teams.Create(ctx, entity.Team{Name: "Team", CreatedBy: ownerID})
	require.NoError(t, err)

	require.NoError(t, members.Add(ctx, teamID, ownerID, valueobject.RoleOwner))

	role, err := members.GetRole(ctx, teamID, ownerID)
	require.NoError(t, err)
	assert.Equal(t, valueobject.RoleOwner, role)

	isMember, err := members.IsMember(ctx, teamID, memberID)
	require.NoError(t, err)
	assert.False(t, isMember)

	require.NoError(t, members.Add(ctx, teamID, memberID, valueobject.RoleMember))
	isMember, err = members.IsMember(ctx, teamID, memberID)
	require.NoError(t, err)
	assert.True(t, isMember)

	list, err := members.ListMembers(ctx, teamID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestRepository_AddDuplicateConflict(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	users := user.New(db)
	teams := team.New(db)
	members := team_member.New(db)
	ctx := context.Background()

	ownerID, err := users.Create(ctx, entity.User{Email: "owner4@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := teams.Create(ctx, entity.Team{Name: "Team2", CreatedBy: ownerID})
	require.NoError(t, err)

	require.NoError(t, members.Add(ctx, teamID, ownerID, valueobject.RoleOwner))
	err = members.Add(ctx, teamID, ownerID, valueobject.RoleAdmin)

	require.Error(t, err)
	assert.Equal(t, 409, terror.HTTPStatus(err))
}

func TestRepository_GetRoleNotFound(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	users := user.New(db)
	teams := team.New(db)
	members := team_member.New(db)
	ctx := context.Background()

	ownerID, err := users.Create(ctx, entity.User{Email: "owner5@example.com", PasswordHash: "h", Name: "Owner"})
	require.NoError(t, err)
	teamID, err := teams.Create(ctx, entity.Team{Name: "Team3", CreatedBy: ownerID})
	require.NoError(t, err)

	_, err = members.GetRole(ctx, teamID, 999999)
	require.Error(t, err)
	assert.Equal(t, 404, terror.HTTPStatus(err))
}
