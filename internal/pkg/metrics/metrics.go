// Package metrics предоставляет счётчики/гистограммы Prometheus и
// middleware для Fiber, которое фиксирует их для каждого запроса.
package metrics

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	ErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_errors_total",
		Help: "Total number of HTTP requests that resulted in a 4xx/5xx.",
	}, []string{"method", "path", "status"})

	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)

// Middleware фиксирует количество запросов, количество ошибок и задержку по каждому маршруту.
func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		status := c.Response().StatusCode()
		path := c.Route().Path
		method := c.Method()
		statusStr := strconv.Itoa(status)

		RequestsTotal.WithLabelValues(method, path, statusStr).Inc()
		RequestDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
		if status >= 400 {
			ErrorsTotal.WithLabelValues(method, path, statusStr).Inc()
		}

		return err
	}
}
