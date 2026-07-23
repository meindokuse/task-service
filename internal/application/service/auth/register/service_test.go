package register_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/meindokuse/task-service/internal/application/service/auth/register"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockUserRepo struct {
	createFn func(ctx context.Context, u entity.User) (uint64, error)
}

func (m *mockUserRepo) Create(ctx context.Context, u entity.User) (uint64, error) {
	return m.createFn(ctx, u)
}

func TestService_Execute_Success(t *testing.T) {
	var captured entity.User
	repo := &mockUserRepo{
		createFn: func(_ context.Context, u entity.User) (uint64, error) {
			captured = u
			return 42, nil
		},
	}

	svc := register.New(repo)
	res, err := svc.Execute(context.Background(), register.Request{
		Email:    "  User@Example.com ",
		Password: "supersecret",
		Name:     " Alice ",
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(42), res.UserID)
	assert.Equal(t, "user@example.com", captured.Email)
	assert.Equal(t, "Alice", captured.Name)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(captured.PasswordHash), []byte("supersecret")))
}

func TestService_Execute_ValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		req  register.Request
	}{
		{"empty email", register.Request{Email: "", Password: "supersecret", Name: "Alice"}},
		{"missing @", register.Request{Email: "not-an-email", Password: "supersecret", Name: "Alice"}},
		{"empty name", register.Request{Email: "a@b.com", Password: "supersecret", Name: "  "}},
		{"short password", register.Request{Email: "a@b.com", Password: "short", Name: "Alice"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockUserRepo{
				createFn: func(context.Context, entity.User) (uint64, error) {
					t.Fatal("repo.Create should not be called on validation failure")
					return 0, nil
				},
			}
			svc := register.New(repo)
			_, err := svc.Execute(context.Background(), tc.req)

			require.Error(t, err)
			assert.Equal(t, 400, terror.HTTPStatus(err))
		})
	}
}

func TestService_Execute_DuplicateEmail(t *testing.T) {
	repo := &mockUserRepo{
		createFn: func(context.Context, entity.User) (uint64, error) {
			return 0, terror.Conflict("email already registered")
		},
	}

	svc := register.New(repo)
	_, err := svc.Execute(context.Background(), register.Request{
		Email:    "a@b.com",
		Password: "supersecret",
		Name:     "Alice",
	})

	require.Error(t, err)
	assert.Equal(t, 409, terror.HTTPStatus(err))
}
