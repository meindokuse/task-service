// Package internal связывает все слои вместе (DI) и владеет жизненным циклом
// процесса. cmd/main.go всегда лишь вызывает internal.New(ctx).Run(ctx).
package internal

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	"github.com/meindokuse/task-service/config"
	"github.com/meindokuse/task-service/internal/pkg/closer"
	"github.com/meindokuse/task-service/internal/pkg/connector"
	"github.com/meindokuse/task-service/internal/pkg/jwtutil"
)

type App struct {
	cfg      *config.Config
	fiberApp *fiber.App
	db       *sqlx.DB
	redis    *redis.Client
	closer   *closer.Closer
}

// New выполняет полную упорядоченную инициализацию: config -> storage ->
// gateways -> services -> server. Фатальные ошибки инициализации
// останавливают процесс прямо здесь, так как cmd/main.go всё равно ничего
// больше с ними не сделает.
func New(ctx context.Context) *App {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := connector.NewMySQL(cfg.MySQL)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}

	redisClient, err := initStorage(ctx, cfg, db)
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}

	cl := closer.New()
	cl.Add(func(context.Context) error { return db.Close() })
	cl.Add(func(context.Context) error { return redisClient.Close() })

	emailClient := initGateways(cfg)
	tokenIssuer := jwtutil.NewIssuer(cfg.JWT.Secret, cfg.JWT.TTL)
	regs := initServices(db, redisClient, tokenIssuer, emailClient)
	fiberApp := initServer(cfg, tokenIssuer, regs)

	return &App{
		cfg:      cfg,
		fiberApp: fiberApp,
		db:       db,
		redis:    redisClient,
		closer:   cl,
	}
}

// Run запускает HTTP-сервер и блокируется до отмены ctx (например, по
// SIGTERM/SIGINT, перехваченному в cmd/main.go), после чего аккуратно
// завершает работу всех компонентов.
func (a *App) Run(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%s", a.cfg.Server.Host, a.cfg.Server.Port)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", addr)
		if err := a.fiberApp.Listen(addr); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := a.fiberApp.ShutdownWithContext(shutdownCtx); err != nil {
		slog.Error("fiber shutdown error", "error", err)
	}

	if err := a.closer.Close(shutdownCtx); err != nil {
		return fmt.Errorf("close resources: %w", err)
	}

	slog.Info("shutdown complete")
	return nil
}
