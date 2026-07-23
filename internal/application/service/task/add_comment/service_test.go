package add_comment_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/task/add_comment"
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
	createFn func(ctx context.Context, c entity.TaskComment) (uint64, error)
}

func (m *mockCommentRepo) Create(ctx context.Context, c entity.TaskComment) (uint64, error) {
	return m.createFn(ctx, c)
}

func TestService_Execute_Success(t *testing.T) {
	var created entity.TaskComment
	tasks := &mockTaskRepo{getByIDFn: func(_ context.Context, id uint64) (*entity.Task, error) {
		return &entity.Task{ID: id, TeamID: 5}, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	comments := &mockCommentRepo{createFn: func(_ context.Context, c entity.TaskComment) (uint64, error) {
		created = c
		return 99, nil
	}}

	svc := add_comment.New(tasks, members, comments)
	res, err := svc.Execute(context.Background(), add_comment.Request{TaskID: 1, UserID: 10, Content: "  looks good  "})

	require.NoError(t, err)
	assert.Equal(t, uint64(99), res.CommentID)
	assert.Equal(t, "looks good", created.Content)
}

func TestService_Execute_EmptyContent(t *testing.T) {
	tasks := &mockTaskRepo{getByIDFn: func(context.Context, uint64) (*entity.Task, error) {
		t.Fatal("should not fetch task")
		return nil, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) {
		t.Fatal("should not check membership")
		return false, nil
	}}
	comments := &mockCommentRepo{createFn: func(context.Context, entity.TaskComment) (uint64, error) {
		t.Fatal("should not create comment")
		return 0, nil
	}}

	svc := add_comment.New(tasks, members, comments)
	_, err := svc.Execute(context.Background(), add_comment.Request{TaskID: 1, UserID: 10, Content: "   "})

	require.Error(t, err)
	assert.Equal(t, 400, terror.HTTPStatus(err))
}

func TestService_Execute_TaskNotFound(t *testing.T) {
	tasks := &mockTaskRepo{getByIDFn: func(context.Context, uint64) (*entity.Task, error) {
		return nil, terror.NotFound("task not found")
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) {
		t.Fatal("should not check membership")
		return false, nil
	}}
	comments := &mockCommentRepo{createFn: func(context.Context, entity.TaskComment) (uint64, error) {
		t.Fatal("should not create comment")
		return 0, nil
	}}

	svc := add_comment.New(tasks, members, comments)
	_, err := svc.Execute(context.Background(), add_comment.Request{TaskID: 1, UserID: 10, Content: "hi"})

	require.Error(t, err)
	assert.Equal(t, 404, terror.HTTPStatus(err))
}

func TestService_Execute_CreateRepoErrorPropagates(t *testing.T) {
	tasks := &mockTaskRepo{getByIDFn: func(_ context.Context, id uint64) (*entity.Task, error) {
		return &entity.Task{ID: id, TeamID: 5}, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return true, nil }}
	comments := &mockCommentRepo{createFn: func(context.Context, entity.TaskComment) (uint64, error) {
		return 0, terror.Internal("insert failed", assert.AnError)
	}}

	svc := add_comment.New(tasks, members, comments)
	_, err := svc.Execute(context.Background(), add_comment.Request{TaskID: 1, UserID: 10, Content: "hi"})

	require.Error(t, err)
	assert.Equal(t, 500, terror.HTTPStatus(err))
}

func TestService_Execute_ForbiddenWhenNotMember(t *testing.T) {
	tasks := &mockTaskRepo{getByIDFn: func(_ context.Context, id uint64) (*entity.Task, error) {
		return &entity.Task{ID: id, TeamID: 5}, nil
	}}
	members := &mockMemberRepo{isMemberFn: func(context.Context, uint64, uint64) (bool, error) { return false, nil }}
	comments := &mockCommentRepo{createFn: func(context.Context, entity.TaskComment) (uint64, error) {
		t.Fatal("should not create comment")
		return 0, nil
	}}

	svc := add_comment.New(tasks, members, comments)
	_, err := svc.Execute(context.Background(), add_comment.Request{TaskID: 1, UserID: 10, Content: "hi"})

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}
