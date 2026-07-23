package list_comments_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/task/list_comments"
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

type mockCommentRepo struct {
	listByTaskFn func(ctx context.Context, taskID uint64) ([]entity.TaskComment, error)
}

func (m *mockCommentRepo) ListByTask(ctx context.Context, taskID uint64) ([]entity.TaskComment, error) {
	return m.listByTaskFn(ctx, taskID)
}

func TestService_Execute_Success(t *testing.T) {
	tasks := &mockTaskRepo{getByIDFn: func(_ context.Context, id uint64) (*entity.Task, error) {
		return &entity.Task{ID: id, TeamID: 5}, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	comments := &mockCommentRepo{listByTaskFn: func(_ context.Context, taskID uint64) ([]entity.TaskComment, error) {
		return []entity.TaskComment{{ID: 1, TaskID: taskID, Content: "hi"}}, nil
	}}

	svc := list_comments.New(tasks, members, comments)
	res, err := svc.Execute(context.Background(), 7, 10)

	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "hi", res[0].Content)
}

func TestService_Execute_ForbiddenWhenNotMember(t *testing.T) {
	tasks := &mockTaskRepo{getByIDFn: func(_ context.Context, id uint64) (*entity.Task, error) {
		return &entity.Task{ID: id, TeamID: 5}, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return false, nil }}
	comments := &mockCommentRepo{listByTaskFn: func(context.Context, uint64) ([]entity.TaskComment, error) {
		t.Fatal("should not list comments")
		return nil, nil
	}}

	svc := list_comments.New(tasks, members, comments)
	_, err := svc.Execute(context.Background(), 7, 10)

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}

func TestService_Execute_TaskNotFound(t *testing.T) {
	tasks := &mockTaskRepo{getByIDFn: func(context.Context, uint64) (*entity.Task, error) {
		return nil, terror.NotFound("task not found")
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) {
		t.Fatal("should not check membership")
		return false, nil
	}}
	comments := &mockCommentRepo{listByTaskFn: func(context.Context, uint64) ([]entity.TaskComment, error) {
		t.Fatal("should not list comments")
		return nil, nil
	}}

	svc := list_comments.New(tasks, members, comments)
	_, err := svc.Execute(context.Background(), 7, 10)

	require.Error(t, err)
	assert.Equal(t, 404, terror.HTTPStatus(err))
}
