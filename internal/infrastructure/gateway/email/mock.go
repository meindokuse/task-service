package email

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// mockClient имитирует нестабильный внешний почтовый провайдер: каждый вызов
// "спит" `latency` и с вероятностью `failureRate` завершается ошибкой, чтобы
// оборачивающему его circuit breaker (см. client.go) было на что реагировать.
type mockClient struct {
	failureRate float64
	latency     time.Duration
}

func newMockClient(failureRate float64, latency time.Duration) *mockClient {
	return &mockClient{failureRate: failureRate, latency: latency}
}

func (m *mockClient) sendInvite(ctx context.Context, toEmail, teamName string) error {
	select {
	case <-time.After(m.latency):
	case <-ctx.Done():
		return ctx.Err()
	}

	if rand.Float64() < m.failureRate {
		return fmt.Errorf("mock email service: delivery to %s failed", toEmail)
	}
	return nil
}
