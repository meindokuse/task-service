// Package register реализует use case регистрации пользователя.
package register

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

const minPasswordLength = 8

type UserRepository interface {
	Create(ctx context.Context, u entity.User) (uint64, error)
}

type Service struct {
	users UserRepository
}

func New(users UserRepository) *Service {
	return &Service{users: users}
}

type Request struct {
	Email    string
	Password string
	Name     string
}

type Response struct {
	UserID uint64
}

func (s *Service) Execute(ctx context.Context, req Request) (Response, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	name := strings.TrimSpace(req.Name)

	if email == "" || !strings.Contains(email, "@") {
		return Response{}, terror.Validation("a valid email is required")
	}
	if name == "" {
		return Response{}, terror.Validation("name is required")
	}
	if len(req.Password) < minPasswordLength {
		return Response{}, terror.Validation("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Response{}, terror.Internal("hash password", err)
	}

	id, err := s.users.Create(ctx, entity.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
	})
	if err != nil {
		return Response{}, err
	}

	return Response{UserID: id}, nil
}
