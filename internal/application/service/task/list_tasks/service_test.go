package list_tasks_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/task/list_tasks"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockTaskRepo struct {
	listFn func(ctx context.Context, f entity.TaskFilter) (entity.TaskList, error)
}

func (m *mockTaskRepo) List(ctx context.Context, f entity.TaskFilter) (entity.TaskList, error) {
	return m.listFn(ctx, f)
}

type mockMemberRepo struct {
	isMemberFn func(ctx context.Context, teamID, userID uint64) (bool, error)
}

func (m *mockMemberRepo) IsMember(ctx context.Context, teamID, userID uint64) (bool, error) {
	return m.isMemberFn(ctx, teamID, userID)
}

type mockCache struct {
	getFn func(ctx context.Context, f entity.TaskFilter) (*entity.TaskList, bool, error)
	setFn func(ctx context.Context, f entity.TaskFilter, list entity.TaskList) error
}

func (m *mockCache) Get(ctx context.Context, f entity.TaskFilter) (*entity.TaskList, bool, error) {
	return m.getFn(ctx, f)
}

func (m *mockCache) Set(ctx context.Context, f entity.TaskFilter, list entity.TaskList) error {
	return m.setFn(ctx, f, list)
}

func TestService_Execute_CacheHitSkipsRepo(t *testing.T) {
	cached := entity.TaskList{Total: 1, Items: []entity.Task{{ID: 1}}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	cache := &mockCache{
		getFn: func(context.Context, entity.TaskFilter) (*entity.TaskList, bool, error) { return &cached, true, nil },
		setFn: func(context.Context, entity.TaskFilter, entity.TaskList) error {
			t.Fatal("should not write cache on hit")
			return nil
		},
	}
	tasks := &mockTaskRepo{listFn: func(context.Context, entity.TaskFilter) (entity.TaskList, error) {
		t.Fatal("should not call repo on cache hit")
		return entity.TaskList{}, nil
	}}

	svc := list_tasks.New(tasks, members, cache)
	res, err := svc.Execute(context.Background(), list_tasks.Request{Filter: entity.TaskFilter{TeamID: 1}, CallerID: 2})

	require.NoError(t, err)
	assert.Equal(t, cached, res)
}

func TestService_Execute_CacheMissFallsBackToRepoAndWritesCache(t *testing.T) {
	fresh := entity.TaskList{Total: 2, Items: []entity.Task{{ID: 1}, {ID: 2}}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	setCalled := false
	cache := &mockCache{
		getFn: func(context.Context, entity.TaskFilter) (*entity.TaskList, bool, error) { return nil, false, nil },
		setFn: func(_ context.Context, _ entity.TaskFilter, list entity.TaskList) error {
			setCalled = true
			assert.Equal(t, fresh, list)
			return nil
		},
	}
	tasks := &mockTaskRepo{listFn: func(context.Context, entity.TaskFilter) (entity.TaskList, error) { return fresh, nil }}

	svc := list_tasks.New(tasks, members, cache)
	res, err := svc.Execute(context.Background(), list_tasks.Request{Filter: entity.TaskFilter{TeamID: 1}, CallerID: 2})

	require.NoError(t, err)
	assert.Equal(t, fresh, res)
	assert.True(t, setCalled)
}

func TestService_Execute_ForbiddenWhenNotMember(t *testing.T) {
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return false, nil }}
	cache := &mockCache{
		getFn: func(context.Context, entity.TaskFilter) (*entity.TaskList, bool, error) {
			t.Fatal("should not touch cache")
			return nil, false, nil
		},
	}
	tasks := &mockTaskRepo{listFn: func(context.Context, entity.TaskFilter) (entity.TaskList, error) {
		t.Fatal("should not call repo")
		return entity.TaskList{}, nil
	}}

	svc := list_tasks.New(tasks, members, cache)
	_, err := svc.Execute(context.Background(), list_tasks.Request{Filter: entity.TaskFilter{TeamID: 1}, CallerID: 2})

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}

func TestService_Execute_CacheReadErrorFallsBackToRepo(t *testing.T) {
	fresh := entity.TaskList{Total: 1, Items: []entity.Task{{ID: 1}}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	cache := &mockCache{
		getFn: func(context.Context, entity.TaskFilter) (*entity.TaskList, bool, error) {
			return nil, false, terror.Internal("redis down", assert.AnError)
		},
		setFn: func(context.Context, entity.TaskFilter, entity.TaskList) error { return nil },
	}
	tasks := &mockTaskRepo{listFn: func(context.Context, entity.TaskFilter) (entity.TaskList, error) { return fresh, nil }}

	svc := list_tasks.New(tasks, members, cache)
	res, err := svc.Execute(context.Background(), list_tasks.Request{Filter: entity.TaskFilter{TeamID: 1}, CallerID: 2})

	require.NoError(t, err)
	assert.Equal(t, fresh, res)
}

func TestService_Execute_RepoErrorPropagates(t *testing.T) {
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	cache := &mockCache{
		getFn: func(context.Context, entity.TaskFilter) (*entity.TaskList, bool, error) { return nil, false, nil },
	}
	tasks := &mockTaskRepo{listFn: func(context.Context, entity.TaskFilter) (entity.TaskList, error) {
		return entity.TaskList{}, terror.Internal("db down", assert.AnError)
	}}

	svc := list_tasks.New(tasks, members, cache)
	_, err := svc.Execute(context.Background(), list_tasks.Request{Filter: entity.TaskFilter{TeamID: 1}, CallerID: 2})

	require.Error(t, err)
	assert.Equal(t, 500, terror.HTTPStatus(err))
}

func TestService_Execute_CacheWriteErrorDoesNotFailRequest(t *testing.T) {
	fresh := entity.TaskList{Total: 1, Items: []entity.Task{{ID: 1}}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	cache := &mockCache{
		getFn: func(context.Context, entity.TaskFilter) (*entity.TaskList, bool, error) { return nil, false, nil },
		setFn: func(context.Context, entity.TaskFilter, entity.TaskList) error {
			return terror.Internal("redis down", assert.AnError)
		},
	}
	tasks := &mockTaskRepo{listFn: func(context.Context, entity.TaskFilter) (entity.TaskList, error) { return fresh, nil }}

	svc := list_tasks.New(tasks, members, cache)
	res, err := svc.Execute(context.Background(), list_tasks.Request{Filter: entity.TaskFilter{TeamID: 1}, CallerID: 2})

	require.NoError(t, err)
	assert.Equal(t, fresh, res)
}
