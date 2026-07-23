// Package list_teams реализует получение списка команд, в которых состоит вызывающий пользователь.
package list_teams

import (
	"context"

	"github.com/meindokuse/task-service/internal/domain/entity"
)

type TeamRepository interface {
	ListForUser(ctx context.Context, userID uint64) ([]entity.Team, error)
}

type Service struct {
	teams TeamRepository
}

func New(teams TeamRepository) *Service {
	return &Service{teams: teams}
}

func (s *Service) Execute(ctx context.Context, userID uint64) ([]entity.Team, error) {
	return s.teams.ListForUser(ctx, userID)
}
