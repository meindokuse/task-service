package login_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/meindokuse/task-service/internal/application/service/auth/login"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockUserRepo struct {
	getByEmailFn func(ctx context.Context, email string) (*entity.User, error)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return m.getByEmailFn(ctx, email)
}

type mockTokenIssuer struct {
	generateFn func(userID uint64) (string, error)
}

func (m *mockTokenIssuer) Generate(userID uint64) (string, error) {
	return m.generateFn(userID)
}

func hashOf(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	return string(hash)
}

func TestService_Execute_Success(t *testing.T) {
	hash := hashOf(t, "supersecret")
	users := &mockUserRepo{
		getByEmailFn: func(_ context.Context, email string) (*entity.User, error) {
			assert.Equal(t, "user@example.com", email)
			return &entity.User{ID: 7, Email: email, PasswordHash: hash}, nil
		},
	}
	tokens := &mockTokenIssuer{
		generateFn: func(userID uint64) (string, error) {
			assert.Equal(t, uint64(7), userID)
			return "signed-token", nil
		},
	}

	svc := login.New(users, tokens)
	res, err := svc.Execute(context.Background(), login.Request{Email: "  User@Example.com ", Password: "supersecret"})

	require.NoError(t, err)
	assert.Equal(t, "signed-token", res.Token)
	assert.Equal(t, uint64(7), res.UserID)
}

func TestService_Execute_UserNotFound(t *testing.T) {
	users := &mockUserRepo{
		getByEmailFn: func(context.Context, string) (*entity.User, error) {
			return nil, terror.NotFound("user not found")
		},
	}
	tokens := &mockTokenIssuer{
		generateFn: func(uint64) (string, error) {
			t.Fatal("token should not be generated")
			return "", nil
		},
	}

	svc := login.New(users, tokens)
	_, err := svc.Execute(context.Background(), login.Request{Email: "nobody@example.com", Password: "whatever"})

	require.Error(t, err)
	assert.Equal(t, 401, terror.HTTPStatus(err))
}

func TestService_Execute_WrongPassword(t *testing.T) {
	hash := hashOf(t, "correct-password")
	users := &mockUserRepo{
		getByEmailFn: func(_ context.Context, email string) (*entity.User, error) {
			return &entity.User{ID: 1, Email: email, PasswordHash: hash}, nil
		},
	}
	tokens := &mockTokenIssuer{
		generateFn: func(uint64) (string, error) {
			t.Fatal("token should not be generated")
			return "", nil
		},
	}

	svc := login.New(users, tokens)
	_, err := svc.Execute(context.Background(), login.Request{Email: "user@example.com", Password: "wrong-password"})

	require.Error(t, err)
	assert.Equal(t, 401, terror.HTTPStatus(err))
}
