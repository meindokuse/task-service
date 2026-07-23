//go:build integration

package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/user"
	"github.com/meindokuse/task-service/internal/pkg/terror"
	"github.com/meindokuse/task-service/internal/testutil/mysqlcontainer"
)

func TestRepository_CreateAndGet(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	repo := user.New(db)
	ctx := context.Background()

	id, err := repo.Create(ctx, entity.User{Email: "alice@example.com", PasswordHash: "hash", Name: "Alice"})
	require.NoError(t, err)
	assert.NotZero(t, id)

	byID, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", byID.Email)
	assert.Equal(t, "Alice", byID.Name)

	byEmail, err := repo.GetByEmail(ctx, "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, id, byEmail.ID)
}

func TestRepository_DuplicateEmailConflict(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	repo := user.New(db)
	ctx := context.Background()

	_, err := repo.Create(ctx, entity.User{Email: "bob@example.com", PasswordHash: "hash", Name: "Bob"})
	require.NoError(t, err)

	_, err = repo.Create(ctx, entity.User{Email: "bob@example.com", PasswordHash: "hash2", Name: "Bob2"})
	require.Error(t, err)
	assert.Equal(t, 409, terror.HTTPStatus(err))
}

func TestRepository_GetByEmailNotFound(t *testing.T) {
	db := mysqlcontainer.Setup(t)
	repo := user.New(db)

	_, err := repo.GetByEmail(context.Background(), "nobody@example.com")
	require.Error(t, err)
	assert.Equal(t, 404, terror.HTTPStatus(err))
}
