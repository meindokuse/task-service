package create_team_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/team/create_team"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockTeamRepo struct {
	createFn func(ctx context.Context, t entity.Team) (uint64, error)
}

func (m *mockTeamRepo) Create(ctx context.Context, t entity.Team) (uint64, error) {
	return m.createFn(ctx, t)
}

type mockMemberRepo struct {
	addFn func(ctx context.Context, teamID, userID uint64, role valueobject.Role) error
}

func (m *mockMemberRepo) Add(ctx context.Context, teamID, userID uint64, role valueobject.Role) error {
	return m.addFn(ctx, teamID, userID, role)
}

type passthroughTx struct{}

func (passthroughTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestService_Execute_Success(t *testing.T) {
	var createdTeam entity.Team
	var addedTeamID, addedUserID uint64
	var addedRole valueobject.Role

	teams := &mockTeamRepo{createFn: func(_ context.Context, tm entity.Team) (uint64, error) {
		createdTeam = tm
		return 99, nil
	}}
	members := &mockMemberRepo{addFn: func(_ context.Context, teamID, userID uint64, role valueobject.Role) error {
		addedTeamID, addedUserID, addedRole = teamID, userID, role
		return nil
	}}

	svc := create_team.New(teams, members, passthroughTx{})
	res, err := svc.Execute(context.Background(), create_team.Request{Name: "  Backend  ", CreatedBy: 5})

	require.NoError(t, err)
	assert.Equal(t, uint64(99), res.TeamID)
	assert.Equal(t, "Backend", createdTeam.Name)
	assert.Equal(t, uint64(5), createdTeam.CreatedBy)
	assert.Equal(t, uint64(99), addedTeamID)
	assert.Equal(t, uint64(5), addedUserID)
	assert.Equal(t, valueobject.RoleOwner, addedRole)
}

func TestService_Execute_EmptyName(t *testing.T) {
	teams := &mockTeamRepo{createFn: func(context.Context, entity.Team) (uint64, error) {
		t.Fatal("should not create team")
		return 0, nil
	}}
	members := &mockMemberRepo{addFn: func(context.Context, uint64, uint64, valueobject.Role) error {
		t.Fatal("should not add member")
		return nil
	}}

	svc := create_team.New(teams, members, passthroughTx{})
	_, err := svc.Execute(context.Background(), create_team.Request{Name: "   ", CreatedBy: 5})

	require.Error(t, err)
	assert.Equal(t, 400, terror.HTTPStatus(err))
}

func TestService_Execute_AddMemberFailurePropagates(t *testing.T) {
	teams := &mockTeamRepo{createFn: func(context.Context, entity.Team) (uint64, error) { return 1, nil }}
	members := &mockMemberRepo{addFn: func(context.Context, uint64, uint64, valueobject.Role) error {
		return terror.Internal("boom", assert.AnError)
	}}

	svc := create_team.New(teams, members, passthroughTx{})
	_, err := svc.Execute(context.Background(), create_team.Request{Name: "Backend", CreatedBy: 5})

	require.Error(t, err)
}
