// Package team_stats предоставляет сложный запрос (a): для каждой команды —
// количество участников и количество завершённых за последние 7 дней задач.
package team_stats

import (
	"context"

	"github.com/meindokuse/task-service/internal/domain/entity"
)

type TeamRepository interface {
	Stats(ctx context.Context) ([]entity.TeamStats, error)
}

type Service struct {
	teams TeamRepository
}

func New(teams TeamRepository) *Service {
	return &Service{teams: teams}
}

func (s *Service) Execute(ctx context.Context) ([]entity.TeamStats, error) {
	return s.teams.Stats(ctx)
}
