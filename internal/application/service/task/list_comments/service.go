// Package list_comments реализует получение списка комментариев задачи, доступное
// только участникам команды, которой принадлежит задача.
package list_comments

import (
	"context"

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
	ListByTask(ctx context.Context, taskID uint64) ([]entity.TaskComment, error)
}

type Service struct {
	tasks    TaskRepository
	members  TeamMemberRepository
	comments CommentRepository
}

func New(tasks TaskRepository, members TeamMemberRepository, comments CommentRepository) *Service {
	return &Service{tasks: tasks, members: members, comments: comments}
}

func (s *Service) Execute(ctx context.Context, taskID, callerID uint64) ([]entity.TaskComment, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	ok, err := s.members.IsMember(ctx, task.TeamID, callerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, terror.Forbidden("you are not a member of this team")
	}

	return s.comments.ListByTask(ctx, taskID)
}
