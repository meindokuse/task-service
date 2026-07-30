// Package create_team реализует создание команды: создатель становится owner.
package create_team

import (
	"context"
	"log/slog"
	"strings"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type TeamRepository interface {
	Create(ctx context.Context, t entity.Team) (uint64, error)
}

type TeamMemberRepository interface {
	Add(ctx context.Context, teamID, userID uint64, role valueobject.Role) error
}

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	teams   TeamRepository
	members TeamMemberRepository
	tx      TxManager
}

func New(teams TeamRepository, members TeamMemberRepository, tx TxManager) *Service {
	return &Service{teams: teams, members: members, tx: tx}
}

type Request struct {
	Name      string
	CreatedBy uint64
}

type Response struct {
	TeamID uint64
}

func (s *Service) Execute(ctx context.Context, req Request) (Response, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Response{}, terror.Validation("team name is required")
	}

	var teamID uint64
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		id, err := s.teams.Create(ctx, entity.Team{Name: name, CreatedBy: req.CreatedBy})
		if err != nil {
			return err
		}
		teamID = id
		return s.members.Add(ctx, id, req.CreatedBy, valueobject.RoleOwner)
	})
	if err != nil {
		return Response{}, err
	}
	slog.Info("team created", "team_id", teamID, "created_by", req.CreatedBy)

	return Response{TeamID: teamID}, nil
}
