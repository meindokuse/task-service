// Package v1 предоставляет HTTP-эндпоинты задач — только преобразование
// DTO<->транспорт, без бизнес-логики. Все маршруты здесь требуют
// аутентифицированного пользователя (см. internal/app/middleware.Auth,
// подключается в internal/init.go).
package v1

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/meindokuse/task-service/internal/app/middleware"
	"github.com/meindokuse/task-service/internal/application/service/task"
	"github.com/meindokuse/task-service/internal/application/service/task/add_comment"
	"github.com/meindokuse/task-service/internal/application/service/task/create_task"
	"github.com/meindokuse/task-service/internal/application/service/task/list_tasks"
	"github.com/meindokuse/task-service/internal/application/service/task/update_task"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
)

type Handler struct {
	registry *task.Registry
}

func New(registry *task.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/tasks", h.createTask)
	router.Get("/tasks", h.listTasks)
	router.Put("/tasks/:id", h.updateTask)
	router.Get("/tasks/:id/history", h.getHistory)
	router.Post("/tasks/:id/comments", h.addComment)
	router.Get("/tasks/:id/comments", h.listComments)
}

func taskID(c *fiber.Ctx) (uint64, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid task id")
	}
	return id, nil
}

type taskResponse struct {
	ID          uint64    `json:"id"`
	TeamID      uint64    `json:"team_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	AssigneeID  *uint64   `json:"assignee_id,omitempty"`
	CreatedBy   uint64    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toTaskResponse(t entity.Task) taskResponse {
	return taskResponse{
		ID:          t.ID,
		TeamID:      t.TeamID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		AssigneeID:  t.AssigneeID,
		CreatedBy:   t.CreatedBy,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

type createTaskRequest struct {
	TeamID      uint64  `json:"team_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	AssigneeID  *uint64 `json:"assignee_id"`
}

func (h *Handler) createTask(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)

	var req createTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	res, err := h.registry.CreateTask.Execute(c.UserContext(), create_task.Request{
		TeamID:      req.TeamID,
		Title:       req.Title,
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
		CreatedBy:   callerID,
	})
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": res.TaskID})
}

type taskListResponse struct {
	Items    []taskResponse `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

func (h *Handler) listTasks(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)

	teamID, err := strconv.ParseUint(c.Query("team_id"), 10, 64)
	if err != nil || teamID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "team_id is required")
	}

	filter := entity.TaskFilter{
		TeamID:   teamID,
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("page_size", 20),
	}

	if status := c.Query("status"); status != "" {
		s := valueobject.TaskStatus(status)
		if !s.Valid() {
			return fiber.NewError(fiber.StatusBadRequest, "invalid status filter")
		}
		filter.Status = &s
	}

	if assignee := c.Query("assignee_id"); assignee != "" {
		id, err := strconv.ParseUint(assignee, 10, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid assignee_id filter")
		}
		filter.AssigneeID = &id
	}

	list, err := h.registry.ListTasks.Execute(c.UserContext(), list_tasks.Request{
		Filter:   filter,
		CallerID: callerID,
	})
	if err != nil {
		return err
	}

	items := make([]taskResponse, 0, len(list.Items))
	for _, t := range list.Items {
		items = append(items, toTaskResponse(t))
	}

	return c.JSON(taskListResponse{Items: items, Total: list.Total, Page: list.Page, PageSize: list.PageSize})
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	AssigneeID  *uint64 `json:"assignee_id"`
	// Unassign позволяет клиенту явно сбросить исполнителя (JSON не различает
	// "поле отсутствует" и "поле равно null" после декодирования в *uint64
	// через обычный map, поэтому для этого случая нужен отдельный булев флаг).
	Unassign bool `json:"unassign"`
}

func (h *Handler) updateTask(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)
	id, err := taskID(c)
	if err != nil {
		return err
	}

	var raw map[string]interface{}
	if err := c.BodyParser(&raw); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	var req updateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	updateReq := update_task.Request{
		TaskID:      id,
		CallerID:    callerID,
		Title:       req.Title,
		Description: req.Description,
	}

	if req.Status != nil {
		s := valueobject.TaskStatus(*req.Status)
		updateReq.Status = &s
	}

	if _, ok := raw["assignee_id"]; ok || req.Unassign {
		updateReq.Assignee = &update_task.AssigneeUpdate{Provided: true, Value: req.AssigneeID}
	}

	updated, err := h.registry.UpdateTask.Execute(c.UserContext(), updateReq)
	if err != nil {
		return err
	}

	return c.JSON(toTaskResponse(updated))
}

type taskHistoryResponse struct {
	ID        uint64    `json:"id"`
	ChangedBy uint64    `json:"changed_by"`
	Field     string    `json:"field"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	ChangedAt time.Time `json:"changed_at"`
}

func (h *Handler) getHistory(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)
	id, err := taskID(c)
	if err != nil {
		return err
	}

	history, err := h.registry.GetHistory.Execute(c.UserContext(), id, callerID)
	if err != nil {
		return err
	}

	resp := make([]taskHistoryResponse, 0, len(history))
	for _, h := range history {
		resp = append(resp, taskHistoryResponse{
			ID:        h.ID,
			ChangedBy: h.ChangedBy,
			Field:     h.Field,
			OldValue:  h.OldValue,
			NewValue:  h.NewValue,
			ChangedAt: h.ChangedAt,
		})
	}
	return c.JSON(resp)
}

type addCommentRequest struct {
	Content string `json:"content"`
}

func (h *Handler) addComment(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)
	id, err := taskID(c)
	if err != nil {
		return err
	}

	var req addCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	res, err := h.registry.AddComment.Execute(c.UserContext(), add_comment.Request{
		TaskID:  id,
		UserID:  callerID,
		Content: req.Content,
	})
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": res.CommentID})
}

type commentResponse struct {
	ID        uint64    `json:"id"`
	UserID    uint64    `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) listComments(c *fiber.Ctx) error {
	callerID, _ := middleware.UserID(c)
	id, err := taskID(c)
	if err != nil {
		return err
	}

	comments, err := h.registry.ListComments.Execute(c.UserContext(), id, callerID)
	if err != nil {
		return err
	}

	resp := make([]commentResponse, 0, len(comments))
	for _, cm := range comments {
		resp = append(resp, commentResponse{ID: cm.ID, UserID: cm.UserID, Content: cm.Content, CreatedAt: cm.CreatedAt})
	}
	return c.JSON(resp)
}
