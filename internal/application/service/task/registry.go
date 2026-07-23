package task

import (
	"github.com/meindokuse/task-service/internal/application/service/task/add_comment"
	"github.com/meindokuse/task-service/internal/application/service/task/create_task"
	"github.com/meindokuse/task-service/internal/application/service/task/get_history"
	"github.com/meindokuse/task-service/internal/application/service/task/list_comments"
	"github.com/meindokuse/task-service/internal/application/service/task/list_tasks"
	"github.com/meindokuse/task-service/internal/application/service/task/update_task"
)

// Registry объединяет все use case'ы задач, создаётся один раз в internal/init.go.
type Registry struct {
	CreateTask   *create_task.Service
	ListTasks    *list_tasks.Service
	UpdateTask   *update_task.Service
	GetHistory   *get_history.Service
	AddComment   *add_comment.Service
	ListComments *list_comments.Service
}

func NewRegistry(
	createTask *create_task.Service,
	listTasks *list_tasks.Service,
	updateTask *update_task.Service,
	getHistory *get_history.Service,
	addComment *add_comment.Service,
	listComments *list_comments.Service,
) *Registry {
	return &Registry{
		CreateTask:   createTask,
		ListTasks:    listTasks,
		UpdateTask:   updateTask,
		GetHistory:   getHistory,
		AddComment:   addComment,
		ListComments: listComments,
	}
}
