package orphaned_assignees_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/team/orphaned_assignees"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockTaskRepo struct {
	orphanedFn func(ctx context.Context, teamID uint64) ([]entity.Task, error)
}

func (m *mockTaskRepo) OrphanedAssignees(ctx context.Context, teamID uint64) ([]entity.Task, error) {
	return m.orphanedFn(ctx, teamID)
}

type mockMemberRepo struct {
	getRoleFn func(ctx context.Context, teamID, userID uint64) (valueobject.Role, error)
}

func (m *mockMemberRepo) GetRole(ctx context.Context, teamID, userID uint64) (valueobject.Role, error) {
	return m.getRoleFn(ctx, teamID, userID)
}

func TestService_Execute_SuccessForOwner(t *testing.T) {
	tasks := &mockTaskRepo{orphanedFn: func(_ context.Context, teamID uint64) ([]entity.Task, error) {
		return []entity.Task{{ID: 1, TeamID: teamID}}, nil
	}}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleOwner, nil
	}}

	svc := orphaned_assignees.New(tasks, members)
	res, err := svc.Execute(context.Background(), 3, 10)

	require.NoError(t, err)
	require.Len(t, res, 1)
}

func TestService_Execute_ForbiddenForPlainMember(t *testing.T) {
	tasks := &mockTaskRepo{orphanedFn: func(context.Context, uint64) ([]entity.Task, error) {
		t.Fatal("should not query")
		return nil, nil
	}}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleMember, nil
	}}

	svc := orphaned_assignees.New(tasks, members)
	_, err := svc.Execute(context.Background(), 3, 10)

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}

func TestService_Execute_ForbiddenWhenNotMember(t *testing.T) {
	tasks := &mockTaskRepo{orphanedFn: func(context.Context, uint64) ([]entity.Task, error) {
		t.Fatal("should not query")
		return nil, nil
	}}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return "", terror.NotFound("not a team member")
	}}

	svc := orphaned_assignees.New(tasks, members)
	_, err := svc.Execute(context.Background(), 3, 10)

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}
