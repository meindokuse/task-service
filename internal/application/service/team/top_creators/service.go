// Package top_creators предоставляет сложный запрос (b): топ-3 автора задач
// за текущий месяц в рамках команды (CTE + оконная функция ROW_NUMBER).
// Просматривать может любой участник команды.
package top_creators

import (
	"context"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type TaskRepository interface {
	TopCreators(ctx context.Context, teamID uint64) ([]entity.TopCreator, error)
}

type TeamMemberRepository interface {
	IsMember(ctx context.Context, teamID, userID uint64) (bool, error)
}

type Service struct {
	tasks   TaskRepository
	members TeamMemberRepository
}

func New(tasks TaskRepository, members TeamMemberRepository) *Service {
	return &Service{tasks: tasks, members: members}
}

func (s *Service) Execute(ctx context.Context, teamID, callerID uint64) ([]entity.TopCreator, error) {
	ok, err := s.members.IsMember(ctx, teamID, callerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, terror.Forbidden("you are not a member of this team")
	}

	return s.tasks.TopCreators(ctx, teamID)
}
