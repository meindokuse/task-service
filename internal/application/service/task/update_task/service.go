// Package update_task реализует обновление задачи: изменять её может только
// создатель, текущий исполнитель или owner/admin команды; каждое изменённое
// поле атомарно вместе с обновлением записывается строкой в task_history.
package update_task

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type TaskRepository interface {
	GetByID(ctx context.Context, id uint64) (*entity.Task, error)
	Update(ctx context.Context, t entity.Task) error
}

type TeamMemberRepository interface {
	GetRole(ctx context.Context, teamID, userID uint64) (valueobject.Role, error)
}

type TaskHistoryRepository interface {
	CreateBatch(ctx context.Context, entries []entity.TaskHistory) error
}

type TaskListCache interface {
	InvalidateTeam(ctx context.Context, teamID uint64) error
}

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	tasks   TaskRepository
	members TeamMemberRepository
	history TaskHistoryRepository
	cache   TaskListCache
	tx      TxManager
}

func New(tasks TaskRepository, members TeamMemberRepository, history TaskHistoryRepository, cache TaskListCache, tx TxManager) *Service {
	return &Service{tasks: tasks, members: members, history: history, cache: cache, tx: tx}
}

// AssigneeUpdate различает "поле не передано" (nil), "передано, снять исполнителя"
// (Provided=true, Value=nil) и "передано, назначить X" (Provided=true, Value=&X).
type AssigneeUpdate struct {
	Provided bool
	Value    *uint64
}

type Request struct {
	TaskID      uint64
	CallerID    uint64
	Title       *string
	Description *string
	Status      *valueobject.TaskStatus
	Assignee    *AssigneeUpdate
}

func (s *Service) Execute(ctx context.Context, req Request) (entity.Task, error) {
	current, err := s.tasks.GetByID(ctx, req.TaskID)
	if err != nil {
		return entity.Task{}, err
	}

	if err := s.authorize(ctx, *current, req.CallerID); err != nil {
		return entity.Task{}, err
	}

	updated := *current
	var entries []entity.TaskHistory

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return entity.Task{}, terror.Validation("title cannot be empty")
		}
		if title != current.Title {
			entries = append(entries, historyEntry(req.TaskID, req.CallerID, "title", current.Title, title))
			updated.Title = title
		}
	}

	if req.Description != nil && *req.Description != current.Description {
		entries = append(entries, historyEntry(req.TaskID, req.CallerID, "description", current.Description, *req.Description))
		updated.Description = *req.Description
	}

	if req.Status != nil {
		if !req.Status.Valid() {
			return entity.Task{}, terror.Validation("invalid status")
		}
		if *req.Status != current.Status {
			entries = append(entries, historyEntry(req.TaskID, req.CallerID, "status", string(current.Status), string(*req.Status)))
			updated.Status = *req.Status
		}
	}

	if req.Assignee != nil && req.Assignee.Provided {
		if !equalAssignee(current.AssigneeID, req.Assignee.Value) {
			entries = append(entries, historyEntry(req.TaskID, req.CallerID, "assignee_id", formatAssignee(current.AssigneeID), formatAssignee(req.Assignee.Value)))
			updated.AssigneeID = req.Assignee.Value
		}
	}

	if len(entries) == 0 {
		return *current, nil
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.tasks.Update(ctx, updated); err != nil {
			return err
		}
		return s.history.CreateBatch(ctx, entries)
	})
	if err != nil {
		return entity.Task{}, err
	}

	if err := s.cache.InvalidateTeam(ctx, updated.TeamID); err != nil {
		slog.Warn("task list cache invalidation failed", "team_id", updated.TeamID, "error", err)
	}

	return updated, nil
}

func (s *Service) authorize(ctx context.Context, task entity.Task, callerID uint64) error {
	if task.CreatedBy == callerID {
		return nil
	}
	if task.AssigneeID != nil && *task.AssigneeID == callerID {
		return nil
	}

	role, err := s.members.GetRole(ctx, task.TeamID, callerID)
	if err != nil {
		var terr *terror.Error
		if errors.As(err, &terr) && terr.Kind == terror.KindNotFound {
			return terror.Forbidden("you cannot update this task")
		}
		return err
	}
	if !role.CanManageMembers() {
		return terror.Forbidden("you cannot update this task")
	}
	return nil
}

func historyEntry(taskID, changedBy uint64, field, oldValue, newValue string) entity.TaskHistory {
	return entity.TaskHistory{
		TaskID:    taskID,
		ChangedBy: changedBy,
		Field:     field,
		OldValue:  oldValue,
		NewValue:  newValue,
	}
}

func equalAssignee(a, b *uint64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func formatAssignee(id *uint64) string {
	if id == nil {
		return ""
	}
	return strconv.FormatUint(*id, 10)
}
