// Package get_history реализует просмотр истории изменений задачи, доступный
// только участникам команды, которой принадлежит задача.
package get_history

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

type TaskHistoryRepository interface {
	ListByTask(ctx context.Context, taskID uint64) ([]entity.TaskHistory, error)
}

type Service struct {
	tasks   TaskRepository
	members TeamMemberRepository
	history TaskHistoryRepository
}

func New(tasks TaskRepository, members TeamMemberRepository, history TaskHistoryRepository) *Service {
	return &Service{tasks: tasks, members: members, history: history}
}

func (s *Service) Execute(ctx context.Context, taskID, callerID uint64) ([]entity.TaskHistory, error) {
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

	return s.history.ListByTask(ctx, taskID)
}
