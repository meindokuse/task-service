// Package add_comment реализует добавление комментария к задаче, доступное
// только участникам команды, которой принадлежит задача.
package add_comment

import (
	"context"
	"log/slog"
	"strings"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type TaskRepository interface {
	GetByID(ctx context.Context, id uint64) (*entity.Task, error)
}

type TeamMemberRepository interface {
	IsMember(ctx context.Context, teamID, userID uint64) (bool, error)
}

type CommentRepository interface {
	Create(ctx context.Context, c entity.TaskComment) (uint64, error)
}

type Service struct {
	tasks    TaskRepository
	members  TeamMemberRepository
	comments CommentRepository
}

func New(tasks TaskRepository, members TeamMemberRepository, comments CommentRepository) *Service {
	return &Service{tasks: tasks, members: members, comments: comments}
}

type Request struct {
	TaskID  uint64
	UserID  uint64
	Content string
}

type Response struct {
	CommentID uint64
}

func (s *Service) Execute(ctx context.Context, req Request) (Response, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return Response{}, terror.Validation("comment content is required")
	}

	task, err := s.tasks.GetByID(ctx, req.TaskID)
	if err != nil {
		return Response{}, err
	}

	ok, err := s.members.IsMember(ctx, task.TeamID, req.UserID)
	if err != nil {
		return Response{}, err
	}
	if !ok {
		return Response{}, terror.Forbidden("you are not a member of this team")
	}

	id, err := s.comments.Create(ctx, entity.TaskComment{
		TaskID:  req.TaskID,
		UserID:  req.UserID,
		Content: content,
	})
	if err != nil {
		return Response{}, err
	}
	slog.Info("comment added", "task_id", req.TaskID, "comment_id", id, "actor_id", req.UserID)

	return Response{CommentID: id}, nil
}
