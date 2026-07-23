// Package list_tasks реализует получение отфильтрованного постраничного списка
// задач, кэшируемого в Redis на 5 минут (TTL задаётся реализацией кэша).
package list_tasks

import (
	"context"
	"log/slog"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type TaskRepository interface {
	List(ctx context.Context, f entity.TaskFilter) (entity.TaskList, error)
}

type TeamMemberRepository interface {
	IsMember(ctx context.Context, teamID, userID uint64) (bool, error)
}

type Cache interface {
	Get(ctx context.Context, f entity.TaskFilter) (*entity.TaskList, bool, error)
	Set(ctx context.Context, f entity.TaskFilter, list entity.TaskList) error
}

type Service struct {
	tasks   TaskRepository
	members TeamMemberRepository
	cache   Cache
}

func New(tasks TaskRepository, members TeamMemberRepository, cache Cache) *Service {
	return &Service{tasks: tasks, members: members, cache: cache}
}

type Request struct {
	Filter   entity.TaskFilter
	CallerID uint64
}

func (s *Service) Execute(ctx context.Context, req Request) (entity.TaskList, error) {
	ok, err := s.members.IsMember(ctx, req.Filter.TeamID, req.CallerID)
	if err != nil {
		return entity.TaskList{}, err
	}
	if !ok {
		return entity.TaskList{}, terror.Forbidden("you are not a member of this team")
	}

	if cached, hit, err := s.cache.Get(ctx, req.Filter); err != nil {
		slog.Warn("task list cache read failed", "team_id", req.Filter.TeamID, "error", err)
	} else if hit {
		return *cached, nil
	}

	list, err := s.tasks.List(ctx, req.Filter)
	if err != nil {
		return entity.TaskList{}, err
	}

	if err := s.cache.Set(ctx, req.Filter, list); err != nil {
		slog.Warn("task list cache write failed", "team_id", req.Filter.TeamID, "error", err)
	}

	return list, nil
}
