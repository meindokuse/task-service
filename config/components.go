package config

import "time"

const (
	ServiceName = "task-service"

	// TaskListCacheTTL — время жизни (TTL) в Redis для кэшированных страниц списка задач.
	TaskListCacheTTL = 5 * time.Minute

	// RateLimitRequestsPerMinute — лимит запросов на одного пользователя.
	RateLimitRequestsPerMinute = 100
	RateLimitWindow            = time.Minute
)
