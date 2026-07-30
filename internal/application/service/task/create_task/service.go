// Package create_task реализует создание задачи, доступное только участникам команды.
package create_task

import (
	"context"
	"log/slog"
	"strings"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type TaskRepository interface {
	Create(ctx context.Context, t entity.Task) (uint64, error)
}

type TeamMemberRepository interface {
	IsMember(ctx context.Context, teamID, userID uint64) (bool, error)
}

type TaskListCache interface {
	InvalidateTeam(ctx context.Context, teamID uint64) error
}

type Service struct {
	tasks   TaskRepository
	members TeamMemberRepository
	cache   TaskListCache
}

func New(tasks TaskRepository, members TeamMemberRepository, cache TaskListCache) *Service {
	return &Service{tasks: tasks, members: members, cache: cache}
}

type Request struct {
	TeamID      uint64
	Title       string
	Description string
	AssigneeID  *uint64
	CreatedBy   uint64
}

type Response struct {
	TaskID uint64
}

func (s *Service) Execute(ctx context.Context, req Request) (Response, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return Response{}, terror.Validation("title is required")
	}

	ok, err := s.members.IsMember(ctx, req.TeamID, req.CreatedBy)
	if err != nil {
		return Response{}, err
	}
	if !ok {
		return Response{}, terror.Forbidden("only team members can create tasks")
	}

	// Примечание: членство исполнителя в команде здесь намеренно НЕ проверяется —
	// запрос целостности "orphaned assignees" (c) как раз и создан для того, чтобы
	// находить задачи, назначенные не-участнику, поэтому это ограничение
	// проверяется там, а не блокируется на запись.
	id, err := s.tasks.Create(ctx, entity.Task{
		TeamID:      req.TeamID,
		Title:       title,
		Description: req.Description,
		Status:      valueobject.TaskStatusTodo,
		AssigneeID:  req.AssigneeID,
		CreatedBy:   req.CreatedBy,
	})
	if err != nil {
		return Response{}, err
	}

	if err := s.cache.InvalidateTeam(ctx, req.TeamID); err != nil {
		slog.Warn("task list cache invalidation failed", "team_id", req.TeamID, "error", err)
	}
	slog.Info("task created", "task_id", id, "team_id", req.TeamID, "created_by", req.CreatedBy)

	return Response{TaskID: id}, nil
}
