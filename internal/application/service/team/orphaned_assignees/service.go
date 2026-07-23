// Package orphaned_assignees предоставляет сложный запрос (c): задачи, чей
// исполнитель не является (или перестал являться) участником команды задачи —
// проверка целостности данных, доступная только owner/admin команды.
package orphaned_assignees

import (
	"context"
	"errors"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type TaskRepository interface {
	OrphanedAssignees(ctx context.Context, teamID uint64) ([]entity.Task, error)
}

type TeamMemberRepository interface {
	GetRole(ctx context.Context, teamID, userID uint64) (valueobject.Role, error)
}

type Service struct {
	tasks   TaskRepository
	members TeamMemberRepository
}

func New(tasks TaskRepository, members TeamMemberRepository) *Service {
	return &Service{tasks: tasks, members: members}
}

func (s *Service) Execute(ctx context.Context, teamID, callerID uint64) ([]entity.Task, error) {
	role, err := s.members.GetRole(ctx, teamID, callerID)
	if err != nil {
		var terr *terror.Error
		if errors.As(err, &terr) && terr.Kind == terror.KindNotFound {
			return nil, terror.Forbidden("you are not a member of this team")
		}
		return nil, err
	}
	if !role.CanManageMembers() {
		return nil, terror.Forbidden("only team owner/admin can view integrity reports")
	}

	return s.tasks.OrphanedAssignees(ctx, teamID)
}
