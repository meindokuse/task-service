// Package email реализует интерфейс EmailGateway (объявленный в use case
// invite_member) как мок-провайдер, обёрнутый в circuit breaker, чтобы
// нестабильный "внешний" почтовый сервис не приводил к каскадным отказам приглашений.
package email

import (
	"context"

	"github.com/sony/gobreaker"

	"github.com/meindokuse/task-service/config"
)

type Client struct {
	inner   *mockClient
	breaker *gobreaker.CircuitBreaker
}

func NewClient(cfg config.EmailConfig) *Client {
	inner := newMockClient(cfg.FailureRate, cfg.Latency)

	settings := gobreaker.Settings{
		Name:        "email-gateway",
		MaxRequests: cfg.BreakerMaxRequests,
		Interval:    cfg.BreakerInterval,
		Timeout:     cfg.BreakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.Requests >= 5 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
		},
	}

	return &Client{inner: inner, breaker: gobreaker.NewCircuitBreaker(settings)}
}

// SendInvite отправляет уведомление о приглашении в команду через circuit
// breaker. Вызывающий код должен трактовать возвращённую ошибку как "сейчас
// не доставлено", а не как повод прерывать весь процесс приглашения.
func (c *Client) SendInvite(ctx context.Context, toEmail, teamName string) error {
	_, err := c.breaker.Execute(func() (interface{}, error) {
		return nil, c.inner.sendInvite(ctx, toEmail, teamName)
	})
	return err
}
