package update_task_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/task/update_task"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockTaskRepo struct {
	task     entity.Task
	updateFn func(ctx context.Context, t entity.Task) error
}

func (m *mockTaskRepo) GetByIDForUpdate(ctx context.Context, id uint64) (*entity.Task, error) {
	task := m.task
	return &task, nil
}

func (m *mockTaskRepo) Update(ctx context.Context, t entity.Task) error {
	return m.updateFn(ctx, t)
}

type mockMemberRepo struct {
	getRoleFn func(ctx context.Context, teamID, userID uint64) (valueobject.Role, error)
}

func (m *mockMemberRepo) GetRole(ctx context.Context, teamID, userID uint64) (valueobject.Role, error) {
	return m.getRoleFn(ctx, teamID, userID)
}

type mockHistoryRepo struct {
	createBatchFn func(ctx context.Context, entries []entity.TaskHistory) error
}

func (m *mockHistoryRepo) CreateBatch(ctx context.Context, entries []entity.TaskHistory) error {
	return m.createBatchFn(ctx, entries)
}

type mockCache struct {
	invalidateFn func(ctx context.Context, teamID uint64) error
}

func (m *mockCache) InvalidateTeam(ctx context.Context, teamID uint64) error {
	return m.invalidateFn(ctx, teamID)
}

type passthroughTx struct{}

func (passthroughTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func baseTask() entity.Task {
	assignee := uint64(20)
	return entity.Task{
		ID:          1,
		TeamID:      3,
		Title:       "Original title",
		Description: "Original description",
		Status:      valueobject.TaskStatusTodo,
		AssigneeID:  &assignee,
		CreatedBy:   10,
	}
}

func TestService_Execute_NoopWhenNothingChanges(t *testing.T) {
	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(context.Context, entity.Task) error {
		t.Fatal("should not update")
		return nil
	}}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleMember, nil
	}}
	history := &mockHistoryRepo{createBatchFn: func(context.Context, []entity.TaskHistory) error {
		t.Fatal("should not write history")
		return nil
	}}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error {
		t.Fatal("should not invalidate cache")
		return nil
	}}

	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	res, err := svc.Execute(context.Background(), update_task.Request{TaskID: 1, CallerID: 10})

	require.NoError(t, err)
	assert.Equal(t, baseTask(), res)
}

func TestService_Execute_TitleChangeByCreatorRecordsHistory(t *testing.T) {
	var updated entity.Task
	var entries []entity.TaskHistory
	invalidatedTeam := uint64(0)

	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(_ context.Context, t entity.Task) error {
		updated = t
		return nil
	}}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleMember, nil
	}}
	history := &mockHistoryRepo{createBatchFn: func(_ context.Context, e []entity.TaskHistory) error {
		entries = e
		return nil
	}}
	cache := &mockCache{invalidateFn: func(_ context.Context, teamID uint64) error {
		invalidatedTeam = teamID
		return nil
	}}

	newTitle := "New title"
	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	res, err := svc.Execute(context.Background(), update_task.Request{
		TaskID: 1, CallerID: 10, // creator
		Title: &newTitle,
	})

	require.NoError(t, err)
	assert.Equal(t, "New title", res.Title)
	assert.Equal(t, "New title", updated.Title)
	require.Len(t, entries, 1)
	assert.Equal(t, "title", entries[0].Field)
	assert.Equal(t, "Original title", entries[0].OldValue)
	assert.Equal(t, "New title", entries[0].NewValue)
	assert.Equal(t, uint64(3), invalidatedTeam)
}

func TestService_Execute_AssigneeCanChangeStatus(t *testing.T) {
	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(context.Context, entity.Task) error { return nil }}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		t.Fatal("assignee should be authorized without a role lookup")
		return "", nil
	}}
	history := &mockHistoryRepo{createBatchFn: func(context.Context, []entity.TaskHistory) error { return nil }}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	done := valueobject.TaskStatusDone
	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	res, err := svc.Execute(context.Background(), update_task.Request{
		TaskID: 1, CallerID: 20, // assignee, not creator
		Status: &done,
	})

	require.NoError(t, err)
	assert.Equal(t, valueobject.TaskStatusDone, res.Status)
}

func TestService_Execute_ForbiddenForUnrelatedMember(t *testing.T) {
	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(context.Context, entity.Task) error {
		t.Fatal("should not update")
		return nil
	}}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleMember, nil
	}}
	history := &mockHistoryRepo{createBatchFn: func(context.Context, []entity.TaskHistory) error {
		t.Fatal("should not write history")
		return nil
	}}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	newTitle := "Hijacked"
	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	_, err := svc.Execute(context.Background(), update_task.Request{
		TaskID: 1, CallerID: 999, // neither creator, assignee, nor admin
		Title: &newTitle,
	})

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}

func TestService_Execute_TeamAdminCanUpdateAnyTask(t *testing.T) {
	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(context.Context, entity.Task) error { return nil }}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleAdmin, nil
	}}
	history := &mockHistoryRepo{createBatchFn: func(context.Context, []entity.TaskHistory) error { return nil }}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	newTitle := "Admin edit"
	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	res, err := svc.Execute(context.Background(), update_task.Request{
		TaskID: 1, CallerID: 500, // team admin, unrelated to the task otherwise
		Title: &newTitle,
	})

	require.NoError(t, err)
	assert.Equal(t, "Admin edit", res.Title)
}

func TestService_Execute_UnassignRecordsHistory(t *testing.T) {
	var entries []entity.TaskHistory
	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(context.Context, entity.Task) error { return nil }}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleMember, nil
	}}
	history := &mockHistoryRepo{createBatchFn: func(_ context.Context, e []entity.TaskHistory) error {
		entries = e
		return nil
	}}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	res, err := svc.Execute(context.Background(), update_task.Request{
		TaskID: 1, CallerID: 10, // creator
		Assignee: &update_task.AssigneeUpdate{Provided: true, Value: nil},
	})

	require.NoError(t, err)
	assert.Nil(t, res.AssigneeID)
	require.Len(t, entries, 1)
	assert.Equal(t, "assignee_id", entries[0].Field)
	assert.Equal(t, "20", entries[0].OldValue)
	assert.Equal(t, "", entries[0].NewValue)
}

func TestService_Execute_InvalidStatusRejected(t *testing.T) {
	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(context.Context, entity.Task) error {
		t.Fatal("should not update")
		return nil
	}}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleMember, nil
	}}
	history := &mockHistoryRepo{createBatchFn: func(context.Context, []entity.TaskHistory) error {
		t.Fatal("should not write history")
		return nil
	}}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	bogus := valueobject.TaskStatus("not-a-real-status")
	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	_, err := svc.Execute(context.Background(), update_task.Request{
		TaskID: 1, CallerID: 10,
		Status: &bogus,
	})

	require.Error(t, err)
	assert.Equal(t, 400, terror.HTTPStatus(err))
}

func TestService_Execute_DescriptionChangeAndReassign(t *testing.T) {
	var updated entity.Task
	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(_ context.Context, t entity.Task) error {
		updated = t
		return nil
	}}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleMember, nil
	}}
	var entries []entity.TaskHistory
	history := &mockHistoryRepo{createBatchFn: func(_ context.Context, e []entity.TaskHistory) error {
		entries = e
		return nil
	}}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	newDesc := "New description"
	newAssignee := uint64(30)
	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	res, err := svc.Execute(context.Background(), update_task.Request{
		TaskID: 1, CallerID: 10, // creator
		Description: &newDesc,
		Assignee:    &update_task.AssigneeUpdate{Provided: true, Value: &newAssignee},
	})

	require.NoError(t, err)
	assert.Equal(t, "New description", res.Description)
	require.NotNil(t, res.AssigneeID)
	assert.Equal(t, uint64(30), *res.AssigneeID)
	assert.Equal(t, "New description", updated.Description)
	require.Len(t, entries, 2)
}

func TestService_Execute_GetByIDErrorPropagates(t *testing.T) {
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		t.Fatal("should not check role")
		return "", nil
	}}
	history := &mockHistoryRepo{createBatchFn: func(context.Context, []entity.TaskHistory) error {
		t.Fatal("should not write history")
		return nil
	}}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	svc := update_task.New(erroringTaskRepo{}, members, history, cache, passthroughTx{})
	_, err := svc.Execute(context.Background(), update_task.Request{TaskID: 1, CallerID: 10})

	require.Error(t, err)
	assert.Equal(t, 404, terror.HTTPStatus(err))
}

type erroringTaskRepo struct{}

func (erroringTaskRepo) GetByIDForUpdate(context.Context, uint64) (*entity.Task, error) {
	return nil, terror.NotFound("task not found")
}

func (erroringTaskRepo) Update(context.Context, entity.Task) error {
	panic("should not be called")
}

func TestService_Execute_RoleLookupErrorPropagates(t *testing.T) {
	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(context.Context, entity.Task) error {
		t.Fatal("should not update")
		return nil
	}}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return "", terror.Internal("db down", assert.AnError)
	}}
	history := &mockHistoryRepo{createBatchFn: func(context.Context, []entity.TaskHistory) error {
		t.Fatal("should not write history")
		return nil
	}}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error { return nil }}

	newTitle := "New title"
	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	_, err := svc.Execute(context.Background(), update_task.Request{
		TaskID: 1, CallerID: 999,
		Title: &newTitle,
	})

	require.Error(t, err)
	assert.Equal(t, 500, terror.HTTPStatus(err))
}

func TestService_Execute_TxErrorPropagates(t *testing.T) {
	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(context.Context, entity.Task) error {
		return terror.Internal("update failed", assert.AnError)
	}}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleMember, nil
	}}
	history := &mockHistoryRepo{createBatchFn: func(context.Context, []entity.TaskHistory) error {
		t.Fatal("should not write history when update fails")
		return nil
	}}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error {
		t.Fatal("should not invalidate cache when tx fails")
		return nil
	}}

	newTitle := "New title"
	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	_, err := svc.Execute(context.Background(), update_task.Request{
		TaskID: 1, CallerID: 10,
		Title: &newTitle,
	})

	require.Error(t, err)
	assert.Equal(t, 500, terror.HTTPStatus(err))
}

func TestService_Execute_CacheInvalidationErrorDoesNotFailRequest(t *testing.T) {
	tasks := &mockTaskRepo{task: baseTask(), updateFn: func(context.Context, entity.Task) error { return nil }}
	members := &mockMemberRepo{getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleMember, nil
	}}
	history := &mockHistoryRepo{createBatchFn: func(context.Context, []entity.TaskHistory) error { return nil }}
	cache := &mockCache{invalidateFn: func(context.Context, uint64) error {
		return terror.Internal("redis down", assert.AnError)
	}}

	newTitle := "New title"
	svc := update_task.New(tasks, members, history, cache, passthroughTx{})
	res, err := svc.Execute(context.Background(), update_task.Request{
		TaskID: 1, CallerID: 10,
		Title: &newTitle,
	})

	require.NoError(t, err)
	assert.Equal(t, "New title", res.Title)
}
