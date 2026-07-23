package auth

import (
	"github.com/meindokuse/task-service/internal/application/service/auth/login"
	"github.com/meindokuse/task-service/internal/application/service/auth/register"
)

// Registry объединяет все use case'ы аутентификации, создаётся один раз в internal/init.go.
type Registry struct {
	Register *register.Service
	Login    *login.Service
}

func NewRegistry(register *register.Service, login *login.Service) *Registry {
	return &Registry{Register: register, Login: login}
}
