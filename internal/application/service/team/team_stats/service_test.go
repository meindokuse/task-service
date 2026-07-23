package team_stats_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/team/team_stats"
	"github.com/meindokuse/task-service/internal/domain/entity"
)

type mockTeamRepo struct {
	statsFn func(ctx context.Context) ([]entity.TeamStats, error)
}

func (m *mockTeamRepo) Stats(ctx context.Context) ([]entity.TeamStats, error) {
	return m.statsFn(ctx)
}

func TestService_Execute_Success(t *testing.T) {
	teams := &mockTeamRepo{statsFn: func(context.Context) ([]entity.TeamStats, error) {
		return []entity.TeamStats{{TeamID: 1, Name: "Backend", MemberCount: 3, DoneLast7Days: 2}}, nil
	}}

	svc := team_stats.New(teams)
	res, err := svc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, 3, res[0].MemberCount)
}
