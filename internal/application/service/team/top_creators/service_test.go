package top_creators_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/team/top_creators"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockTaskRepo struct {
	topCreatorsFn func(ctx context.Context, teamID uint64) ([]entity.TopCreator, error)
}

func (m *mockTaskRepo) TopCreators(ctx context.Context, teamID uint64) ([]entity.TopCreator, error) {
	return m.topCreatorsFn(ctx, teamID)
}

type mockMemberRepo struct {
	isMemberFn func(ctx context.Context, teamID, userID uint64) (bool, error)
}

func (m *mockMemberRepo) IsMember(ctx context.Context, teamID, userID uint64) (bool, error) {
	return m.isMemberFn(ctx, teamID, userID)
}

func TestService_Execute_Success(t *testing.T) {
	tasks := &mockTaskRepo{topCreatorsFn: func(_ context.Context, teamID uint64) ([]entity.TopCreator, error) {
		return []entity.TopCreator{{TeamID: teamID, UserID: 1, TasksCount: 5, RankInTeam: 1}}, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}

	svc := top_creators.New(tasks, members)
	res, err := svc.Execute(context.Background(), 3, 10)

	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, 1, res[0].RankInTeam)
}

func TestService_Execute_ForbiddenWhenNotMember(t *testing.T) {
	tasks := &mockTaskRepo{topCreatorsFn: func(context.Context, uint64) ([]entity.TopCreator, error) {
		t.Fatal("should not query top creators")
		return nil, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return false, nil }}

	svc := top_creators.New(tasks, members)
	_, err := svc.Execute(context.Background(), 3, 10)

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}
