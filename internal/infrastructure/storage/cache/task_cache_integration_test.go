//go:build integration

package cache_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/cache"
)

func setupRedis(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "6379/tcp")
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", host, port.Port())})
	t.Cleanup(func() { _ = client.Close() })

	require.NoError(t, client.Ping(ctx).Err())
	return client
}

func TestTaskListCache_SetGetInvalidate(t *testing.T) {
	client := setupRedis(t)
	c := cache.NewTaskListCache(client, time.Minute)
	ctx := context.Background()

	filter := entity.TaskFilter{TeamID: 1, Page: 1, PageSize: 20}

	_, hit, err := c.Get(ctx, filter)
	require.NoError(t, err)
	assert.False(t, hit)

	list := entity.TaskList{Total: 1, Page: 1, PageSize: 20, Items: []entity.Task{{ID: 1, Title: "Task"}}}
	require.NoError(t, c.Set(ctx, filter, list))

	got, hit, err := c.Get(ctx, filter)
	require.NoError(t, err)
	require.True(t, hit)
	assert.Equal(t, list.Total, got.Total)
	require.Len(t, got.Items, 1)
	assert.Equal(t, "Task", got.Items[0].Title)

	require.NoError(t, c.InvalidateTeam(ctx, 1))

	_, hit, err = c.Get(ctx, filter)
	require.NoError(t, err)
	assert.False(t, hit)
}

func TestTaskListCache_DifferentFiltersAreDistinctKeys(t *testing.T) {
	client := setupRedis(t)
	c := cache.NewTaskListCache(client, time.Minute)
	ctx := context.Background()

	statusDone := valueobject.TaskStatusDone
	filterAll := entity.TaskFilter{TeamID: 2, Page: 1, PageSize: 20}
	filterDone := entity.TaskFilter{TeamID: 2, Status: &statusDone, Page: 1, PageSize: 20}

	require.NoError(t, c.Set(ctx, filterAll, entity.TaskList{Total: 5}))

	_, hit, err := c.Get(ctx, filterDone)
	require.NoError(t, err)
	assert.False(t, hit, "a different filter must not share the cache entry")
}
