package list_teams_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/team/list_teams"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockTeamRepo struct {
	listForUserFn func(ctx context.Context, userID uint64) ([]entity.Team, error)
}

func (m *mockTeamRepo) ListForUser(ctx context.Context, userID uint64) ([]entity.Team, error) {
	return m.listForUserFn(ctx, userID)
}

func TestService_Execute_Success(t *testing.T) {
	teams := &mockTeamRepo{listForUserFn: func(_ context.Context, userID uint64) ([]entity.Team, error) {
		return []entity.Team{{ID: 1, Name: "Backend", CreatedBy: userID}}, nil
	}}

	svc := list_teams.New(teams)
	res, err := svc.Execute(context.Background(), 5)

	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "Backend", res[0].Name)
}

func TestService_Execute_RepoErrorPropagates(t *testing.T) {
	teams := &mockTeamRepo{listForUserFn: func(context.Context, uint64) ([]entity.Team, error) {
		return nil, terror.Internal("db down", assert.AnError)
	}}

	svc := list_teams.New(teams)
	_, err := svc.Execute(context.Background(), 5)

	require.Error(t, err)
	assert.Equal(t, 500, terror.HTTPStatus(err))
}
