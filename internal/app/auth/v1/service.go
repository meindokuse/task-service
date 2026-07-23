// Package v1 предоставляет HTTP-эндпоинты аутентификации (register/login) —
// только преобразование DTO<->транспорт, без бизнес-логики.
package v1

import (
	"github.com/gofiber/fiber/v2"

	"github.com/meindokuse/task-service/internal/application/service/auth"
	"github.com/meindokuse/task-service/internal/application/service/auth/login"
	"github.com/meindokuse/task-service/internal/application/service/auth/register"
)

type Handler struct {
	registry *auth.Registry
}

func New(registry *auth.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/register", h.register)
	router.Post("/login", h.login)
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type registerResponse struct {
	UserID uint64 `json:"user_id"`
}

func (h *Handler) register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	res, err := h.registry.Register.Execute(c.UserContext(), register.Request{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(registerResponse{UserID: res.UserID})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token  string `json:"token"`
	UserID uint64 `json:"user_id"`
}

func (h *Handler) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	res, err := h.registry.Login.Execute(c.UserContext(), login.Request{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	return c.JSON(loginResponse{Token: res.Token, UserID: res.UserID})
}
