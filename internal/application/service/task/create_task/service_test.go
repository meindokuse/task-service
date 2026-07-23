package create_task_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/task/create_task"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockTaskRepo struct {
	createFn func(ctx context.Context, t entity.Task) (uint64, error)
}

func (m *mockTaskRepo) Create(ctx context.Context, t entity.Task) (uint64, error) {
	return m.createFn(ctx, t)
}

type mockMemberRepo struct {
	isMemberFn func(ctx context.Context, teamID, userID uint64) (bool, error)
}

func (m *mockMemberRepo) IsMember(ctx context.Context, teamID, userID uint64) (bool, error) {
	return m.isMemberFn(ctx, teamID, userID)
}

type mockCache struct {
	invalidateFn func(ctx context.Context, teamID uint64) error
}

func (m *mockCache) InvalidateTeam(ctx context.Context, teamID uint64) error {
	return m.invalidateFn(ctx, teamID)
}

func TestService_Execute_Success(t *testing.T) {
	var created entity.Task
	invalidated := false

	tasks := &mockTaskRepo{createFn: func(_ context.Context, tk entity.Task) (uint64, error) {
		created = tk
		return 55, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	cache := &mockCache{invalidateFn: func(_ context.Context, teamID uint64) error {
		invalidated = true
		assert.Equal(t, uint64(3), teamID)
		return nil
	}}

	svc := create_task.New(tasks, members, cache)
	res, err := svc.Execute(context.Background(), create_task.Request{
		TeamID: 3, Title: "  Write tests  ", Description: "desc", CreatedBy: 7,
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(55), res.TaskID)
	assert.Equal(t, "Write tests", created.Title)
	assert.Equal(t, valueobject.TaskStatusTodo, created.Status)
	assert.True(t, invalidated)
}

func TestService_Execute_EmptyTitle(t *testing.T) {
	tasks := &mockTaskRepo{createFn: func(context.Context, entity.Task) (uint64, error) {
		t.Fatal("should not create task")
		return 0, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	svc := create_task.New(tasks, members, cache)
	_, err := svc.Execute(context.Background(), create_task.Request{TeamID: 3, Title: "   ", CreatedBy: 7})

	require.Error(t, err)
	assert.Equal(t, 400, terror.HTTPStatus(err))
}

func TestService_Execute_ForbiddenWhenNotMember(t *testing.T) {
	tasks := &mockTaskRepo{createFn: func(context.Context, entity.Task) (uint64, error) {
		t.Fatal("should not create task")
		return 0, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return false, nil }}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	svc := create_task.New(tasks, members, cache)
	_, err := svc.Execute(context.Background(), create_task.Request{TeamID: 3, Title: "Task", CreatedBy: 7})

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}

func TestService_Execute_MembershipCheckErrorPropagates(t *testing.T) {
	tasks := &mockTaskRepo{createFn: func(context.Context, entity.Task) (uint64, error) {
		t.Fatal("should not create task")
		return 0, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) {
		return false, terror.Internal("db down", assert.AnError)
	}}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	svc := create_task.New(tasks, members, cache)
	_, err := svc.Execute(context.Background(), create_task.Request{TeamID: 3, Title: "Task", CreatedBy: 7})

	require.Error(t, err)
	assert.Equal(t, 500, terror.HTTPStatus(err))
}

func TestService_Execute_CreateRepoErrorPropagates(t *testing.T) {
	tasks := &mockTaskRepo{createFn: func(context.Context, entity.Task) (uint64, error) {
		return 0, terror.Internal("insert failed", assert.AnError)
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error {
		t.Fatal("should not invalidate cache when create fails")
		return nil
	}}

	svc := create_task.New(tasks, members, cache)
	_, err := svc.Execute(context.Background(), create_task.Request{TeamID: 3, Title: "Task", CreatedBy: 7})

	require.Error(t, err)
	assert.Equal(t, 500, terror.HTTPStatus(err))
}

func TestService_Execute_CacheInvalidationErrorDoesNotFailRequest(t *testing.T) {
	tasks := &mockTaskRepo{createFn: func(context.Context, entity.Task) (uint64, error) { return 1, nil }}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error {
		return terror.Internal("redis down", assert.AnError)
	}}

	svc := create_task.New(tasks, members, cache)
	res, err := svc.Execute(context.Background(), create_task.Request{TeamID: 3, Title: "Task", CreatedBy: 7})

	require.NoError(t, err)
	assert.Equal(t, uint64(1), res.TaskID)
}
