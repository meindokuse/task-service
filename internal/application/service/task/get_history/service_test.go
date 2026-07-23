package get_history_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/task/get_history"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockTaskRepo struct {
	getByIDFn func(ctx context.Context, id uint64) (*entity.Task, error)
}

func (m *mockTaskRepo) GetByID(ctx context.Context, id uint64) (*entity.Task, error) {
	return m.getByIDFn(ctx, id)
}

type mockMemberRepo struct {
	isMemberFn func(ctx context.Context, teamID, userID uint64) (bool, error)
}

func (m *mockMemberRepo) IsMember(ctx context.Context, teamID, userID uint64) (bool, error) {
	return m.isMemberFn(ctx, teamID, userID)
}

type mockHistoryRepo struct {
	listByTaskFn func(ctx context.Context, taskID uint64) ([]entity.TaskHistory, error)
}

func (m *mockHistoryRepo) ListByTask(ctx context.Context, taskID uint64) ([]entity.TaskHistory, error) {
	return m.listByTaskFn(ctx, taskID)
}

func TestService_Execute_Success(t *testing.T) {
	tasks := &mockTaskRepo{getByIDFn: func(_ context.Context, id uint64) (*entity.Task, error) {
		return &entity.Task{ID: id, TeamID: 5}, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	history := &mockHistoryRepo{listByTaskFn: func(_ context.Context, taskID uint64) ([]entity.TaskHistory, error) {
		return []entity.TaskHistory{{ID: 1, TaskID: taskID, Field: "status"}}, nil
	}}

	svc := get_history.New(tasks, members, history)
	res, err := svc.Execute(context.Background(), 7, 10)

	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "status", res[0].Field)
}

func TestService_Execute_TaskNotFound(t *testing.T) {
	tasks := &mockTaskRepo{getByIDFn: func(context.Context, uint64) (*entity.Task, error) {
		return nil, terror.NotFound("task not found")
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) {
		t.Fatal("should not check membership")
		return false, nil
	}}
	history := &mockHistoryRepo{listByTaskFn: func(context.Context, uint64) ([]entity.TaskHistory, error) {
		t.Fatal("should not list history")
		return nil, nil
	}}

	svc := get_history.New(tasks, members, history)
	_, err := svc.Execute(context.Background(), 7, 10)

	require.Error(t, err)
	assert.Equal(t, 404, terror.HTTPStatus(err))
}

func TestService_Execute_ForbiddenWhenNotMember(t *testing.T) {
	tasks := &mockTaskRepo{getByIDFn: func(_ context.Context, id uint64) (*entity.Task, error) {
		return &entity.Task{ID: id, TeamID: 5}, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return false, nil }}
	history := &mockHistoryRepo{listByTaskFn: func(context.Context, uint64) ([]entity.TaskHistory, error) {
		t.Fatal("should not list history")
		return nil, nil
	}}

	svc := get_history.New(tasks, members, history)
	_, err := svc.Execute(context.Background(), 7, 10)

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}
