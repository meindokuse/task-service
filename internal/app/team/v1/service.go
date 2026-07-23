// Package v1 предоставляет HTTP-эндпоинты команд — только преобразование
// DTO<->транспорт, без бизнес-логики. Все маршруты здесь требуют
// аутентифицированного пользователя (см. internal/app/middleware.Auth,
// подключается в internal/init.go).
package v1

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/meindokuse/task-service/internal/app/middleware"
	"github.com/meindokuse/task-service/internal/application/service/team"
	"github.com/meindokuse/task-service/internal/application/service/team/create_team"
	"github.com/meindokuse/task-service/internal/application/service/team/invite_member"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
)

type Handler struct {
	registry *team.Registry
}

func New(registry *team.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/teams", h.createTeam)
	router.Get("/teams", h.listTeams)
	router.Post("/teams/:id/invite", h.inviteMember)
	router.Get("/teams/stats", h.stats)
	router.Get("/teams/:id/top-creators", h.topCreators)
	router.Get("/teams/:id/orphaned-assignees", h.orphanedAssignees)
}

func teamID(c *fiber.Ctx) (uint64, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid team id")
	}
	return id, nil
}

type createTeamRequest struct {
	Name string `json:"name"`
}

type teamResponse struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name,omitempty"`
	CreatedBy uint64    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

func (h *Handler) createTeam(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)

	var req createTeamRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	res, err := h.registry.CreateTeam.Execute(c.UserContext(), create_team.Request{
		Name:      req.Name,
		CreatedBy: callerID,
	})
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": res.TeamID})
}

func (h *Handler) listTeams(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)

	teams, err := h.registry.ListTeams.Execute(c.UserContext(), callerID)
	if err != nil {
		return err
	}

	resp := make([]teamResponse, 0, len(teams))
	for _, t := range teams {
		resp = append(resp, teamResponse{ID: t.ID, Name: t.Name, CreatedBy: t.CreatedBy, CreatedAt: t.CreatedAt})
	}
	return c.JSON(resp)
}

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type inviteResponse struct {
	UserID         uint64 `json:"user_id"`
	EmailDelivered bool   `json:"email_delivered"`
}

func (h *Handler) inviteMember(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)
	id, err := teamID(c)
	if err != nil {
		return err
	}

	var req inviteRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	res, err := h.registry.InviteMember.Execute(c.UserContext(), invite_member.Request{
		TeamID:       id,
		InviterID:    callerID,
		InviteeEmail: req.Email,
		Role:         valueobject.Role(req.Role),
	})
	if err != nil {
		return err
	}

	return c.JSON(inviteResponse{UserID: res.UserID, EmailDelivered: res.EmailDelivered})
}

type teamStatsResponse struct {
	TeamID        uint64 `json:"team_id"`
	Name          string `json:"name"`
	MemberCount   int    `json:"member_count"`
	DoneLast7Days int    `json:"done_last_7_days"`
}

func (h *Handler) stats(c *fiber.Ctx) error {
	stats, err := h.registry.TeamStats.Execute(c.UserContext())
	if err != nil {
		return err
	}

	resp := make([]teamStatsResponse, 0, len(stats))
	for _, s := range stats {
		resp = append(resp, teamStatsResponse{
			TeamID:        s.TeamID,
			Name:          s.Name,
			MemberCount:   s.MemberCount,
			DoneLast7Days: s.DoneLast7Days,
		})
	}
	return c.JSON(resp)
}

type topCreatorResponse struct {
	UserID     uint64 `json:"user_id"`
	UserName   string `json:"user_name"`
	TasksCount int    `json:"tasks_count"`
	Rank       int    `json:"rank"`
}

func (h *Handler) topCreators(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)
	id, err := teamID(c)
	if err != nil {
		return err
	}

	creators, err := h.registry.TopCreators.Execute(c.UserContext(), id, callerID)
	if err != nil {
		return err
	}

	resp := make([]topCreatorResponse, 0, len(creators))
	for _, cr := range creators {
		resp = append(resp, topCreatorResponse{
			UserID:     cr.UserID,
			UserName:   cr.UserName,
			TasksCount: cr.TasksCount,
			Rank:       cr.RankInTeam,
		})
	}
	return c.JSON(resp)
}

type orphanedTaskResponse struct {
	ID         uint64 `json:"id"`
	Title      string `json:"title"`
	AssigneeID uint64 `json:"assignee_id"`
}

func (h *Handler) orphanedAssignees(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)
	id, err := teamID(c)
	if err != nil {
		return err
	}

	tasks, err := h.registry.OrphanedAssignees.Execute(c.UserContext(), id, callerID)
	if err != nil {
		return err
	}

	resp := make([]orphanedTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		var assigneeID uint64
		if t.AssigneeID != nil {
			assigneeID = *t.AssigneeID
		}
		resp = append(resp, orphanedTaskResponse{ID: t.ID, Title: t.Title, AssigneeID: assigneeID})
	}
	return c.JSON(resp)
}
