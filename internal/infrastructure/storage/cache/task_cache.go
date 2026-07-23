// Package cache реализует кэш списка задач на базе Redis, используемый use
// case'ом list_tasks: ключ строится из команды+фильтров+страницы, TTL 5 минут,
// инвалидируется при любой записи (создании/обновлении) задачи в этой команде.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type TaskListCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewTaskListCache(client *redis.Client, ttl time.Duration) *TaskListCache {
	return &TaskListCache{client: client, ttl: ttl}
}

func (c *TaskListCache) key(f entity.TaskFilter) string {
	status := "any"
	if f.Status != nil {
		status = string(*f.Status)
	}
	assignee := "any"
	if f.AssigneeID != nil {
		assignee = strconv.FormatUint(*f.AssigneeID, 10)
	}
	return fmt.Sprintf("tasks:list:%d:%s:%s:%d:%d", f.TeamID, status, assignee, f.Page, f.PageSize)
}

// Get возвращает закэшированную страницу и true, либо (nil, false) при промахе кэша.
func (c *TaskListCache) Get(ctx context.Context, f entity.TaskFilter) (*entity.TaskList, bool, error) {
	val, err := c.client.Get(ctx, c.key(f)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, terror.Internal("read task list cache", err)
	}

	var list entity.TaskList
	if err := json.Unmarshal([]byte(val), &list); err != nil {
		return nil, false, terror.Internal("decode task list cache", err)
	}
	return &list, true, nil
}

func (c *TaskListCache) Set(ctx context.Context, f entity.TaskFilter, list entity.TaskList) error {
	data, err := json.Marshal(list)
	if err != nil {
		return terror.Internal("encode task list cache", err)
	}
	if err := c.client.Set(ctx, c.key(f), data, c.ttl).Err(); err != nil {
		return terror.Internal("write task list cache", err)
	}
	return nil
}

// InvalidateTeam сбрасывает все закэшированные страницы команды; вызывается
// после любого создания/обновления задачи в этой команде.
func (c *TaskListCache) InvalidateTeam(ctx context.Context, teamID uint64) error {
	pattern := fmt.Sprintf("tasks:list:%d:*", teamID)

	var keys []string
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return terror.Internal("scan task list cache", err)
	}
	if len(keys) == 0 {
		return nil
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return terror.Internal("invalidate task list cache", err)
	}
	return nil
}
