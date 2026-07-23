// Package login реализует use case аутентификации (выпуск JWT-токена).
package login

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
}

type TokenIssuer interface {
	Generate(userID uint64) (string, error)
}

type Service struct {
	users  UserRepository
	tokens TokenIssuer
}

func New(users UserRepository, tokens TokenIssuer) *Service {
	return &Service{users: users, tokens: tokens}
}

type Request struct {
	Email    string
	Password string
}

type Response struct {
	Token  string
	UserID uint64
}

var errInvalidCredentials = terror.Unauthorized("invalid email or password")

func (s *Service) Execute(ctx context.Context, req Request) (Response, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		var terr *terror.Error
		if errors.As(err, &terr) && terr.Kind == terror.KindNotFound {
			return Response{}, errInvalidCredentials
		}
		return Response{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return Response{}, errInvalidCredentials
	}

	token, err := s.tokens.Generate(u.ID)
	if err != nil {
		return Response{}, terror.Internal("issue token", err)
	}

	return Response{Token: token, UserID: u.ID}, nil
}
