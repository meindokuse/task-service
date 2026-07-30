package email

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/config"
)

// TestClient_OnStateChangeLogsBreakerTrip forces enough failures to trip the
// circuit breaker and asserts the OnStateChange callback logs the
// closed->open transition, so breaker state changes are no longer invisible.
func TestClient_OnStateChangeLogsBreakerTrip(t *testing.T) {
	var buf bytes.Buffer
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prevLogger)

	client := NewClient(config.EmailConfig{
		FailureRate:        1.0,
		Latency:            0,
		BreakerMaxRequests: 1,
		BreakerInterval:    time.Minute,
		BreakerTimeout:     time.Minute,
	})

	for i := 0; i < 5; i++ {
		err := client.SendInvite(context.Background(), "user@example.com", "Team")
		require.Error(t, err)
	}

	logs := buf.String()
	assert.Contains(t, logs, "circuit breaker state change")
	assert.Contains(t, logs, "breaker=email-gateway")
	assert.Contains(t, logs, "from=closed")
	assert.Contains(t, logs, "to=open")
}
