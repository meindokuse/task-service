package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/meindokuse/task-service/config"
	authv1 "github.com/meindokuse/task-service/internal/app/auth/v1"
	appmw "github.com/meindokuse/task-service/internal/app/middleware"
	taskv1 "github.com/meindokuse/task-service/internal/app/task/v1"
	teamv1 "github.com/meindokuse/task-service/internal/app/team/v1"
	"github.com/meindokuse/task-service/internal/application/service/auth"
	"github.com/meindokuse/task-service/internal/application/service/auth/login"
	"github.com/meindokuse/task-service/internal/application/service/auth/register"
	"github.com/meindokuse/task-service/internal/application/service/task"
	"github.com/meindokuse/task-service/internal/application/service/task/add_comment"
	"github.com/meindokuse/task-service/internal/application/service/task/create_task"
	"github.com/meindokuse/task-service/internal/application/service/task/get_history"
	"github.com/meindokuse/task-service/internal/application/service/task/list_comments"
	"github.com/meindokuse/task-service/internal/application/service/task/list_tasks"
	"github.com/meindokuse/task-service/internal/application/service/task/update_task"
	"github.com/meindokuse/task-service/internal/application/service/team"
	"github.com/meindokuse/task-service/internal/application/service/team/create_team"
	"github.com/meindokuse/task-service/internal/application/service/team/invite_member"
	"github.com/meindokuse/task-service/internal/application/service/team/list_teams"
	"github.com/meindokuse/task-service/internal/application/service/team/orphaned_assignees"
	"github.com/meindokuse/task-service/internal/application/service/team/team_stats"
	"github.com/meindokuse/task-service/internal/application/service/team/top_creators"
	"github.com/meindokuse/task-service/internal/infrastructure/gateway/email"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/cache"
	taskstorage "github.com/meindokuse/task-service/internal/infrastructure/storage/task"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/task_comment"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/task_history"
	teamstorage "github.com/meindokuse/task-service/internal/infrastructure/storage/team"
	"github.com/meindokuse/task-service/internal/infrastructure/storage/team_member"
	userstorage "github.com/meindokuse/task-service/internal/infrastructure/storage/user"
	"github.com/meindokuse/task-service/internal/pkg/connector"
	"github.com/meindokuse/task-service/internal/pkg/jwtutil"
	"github.com/meindokuse/task-service/internal/pkg/metrics"
	"github.com/meindokuse/task-service/internal/pkg/txmanager"
	"github.com/meindokuse/task-service/migrations"
)

// initStorage открывает пул MySQL, накатывает на него неприменённые миграции
// и открывает клиент Redis — именно в таком порядке, так как всё, что идёт
// дальше, требует уже мигрированной схемы.
func initStorage(ctx context.Context, cfg *config.Config, db *sqlx.DB) (*redis.Client, error) {
	if err := runMigrations(db.DB, cfg.MySQL.Database); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	redisClient, err := connector.NewRedis(ctx, cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}

	return redisClient, nil
}

func runMigrations(db *sql.DB, databaseName string) error {
	driver, err := mysqlmigrate.WithInstance(db, &mysqlmigrate.Config{})
	if err != nil {
		return fmt.Errorf("mysql migrate driver: %w", err)
	}

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, databaseName, driver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func initGateways(cfg *config.Config) *email.Client {
	return email.NewClient(cfg.Email)
}

// registries объединяет три реестра use case'ов (по одному на домен),
// созданных здесь и используемых в initServer.
type registries struct {
	auth *auth.Registry
	team *team.Registry
	task *task.Registry
}

func initServices(db *sqlx.DB, redisClient *redis.Client, tokenIssuer *jwtutil.Issuer, emailClient *email.Client) registries {
	tx := txmanager.New(db)

	userRepo := userstorage.New(db)
	teamRepo := teamstorage.New(db)
	teamMemberRepo := team_member.New(db)
	taskRepo := taskstorage.New(db)
	taskHistoryRepo := task_history.New(db)
	taskCommentRepo := task_comment.New(db)
	taskListCache := cache.NewTaskListCache(redisClient, config.TaskListCacheTTL)

	authRegistry := auth.NewRegistry(
		register.New(userRepo),
		login.New(userRepo, tokenIssuer),
	)

	teamRegistry := team.NewRegistry(
		create_team.New(teamRepo, teamMemberRepo, tx),
		list_teams.New(teamRepo),
		invite_member.New(teamRepo, teamMemberRepo, userRepo, emailClient),
		team_stats.New(teamRepo),
		top_creators.New(taskRepo, teamMemberRepo),
		orphaned_assignees.New(taskRepo, teamMemberRepo),
	)

	taskRegistry := task.NewRegistry(
		create_task.New(taskRepo, teamMemberRepo, taskListCache),
		list_tasks.New(taskRepo, teamMemberRepo, taskListCache),
		update_task.New(taskRepo, teamMemberRepo, taskHistoryRepo, taskListCache, tx),
		get_history.New(taskRepo, teamMemberRepo, taskHistoryRepo),
		add_comment.New(taskRepo, teamMemberRepo, taskCommentRepo),
		list_comments.New(taskRepo, teamMemberRepo, taskCommentRepo),
	)

	return registries{auth: authRegistry, team: teamRegistry, task: taskRegistry}
}

// initServer собирает Fiber-приложение: глобальные middleware, публичные
// маршруты, затем группа аутентифицированных маршрутов с rate limiting.
func initServer(cfg *config.Config, tokenIssuer *jwtutil.Issuer, regs registries) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler:          appmw.ErrorHandler,
		DisableStartupMessage: true,
	})

	app.Use(recover.New())
	app.Use(metrics.Middleware())

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	public := app.Group("/api/v1", appmw.RateLimitByIP())
	authv1.New(regs.auth).RegisterRoutes(public)

	protected := app.Group("/api/v1", appmw.Auth(tokenIssuer), appmw.RateLimitByUser())
	teamv1.New(regs.team).RegisterRoutes(protected)
	taskv1.New(regs.task).RegisterRoutes(protected)

	return app
}
