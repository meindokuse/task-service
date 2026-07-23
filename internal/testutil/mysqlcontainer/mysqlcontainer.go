//go:build integration

// Package mysqlcontainer поднимает настоящий контейнер MySQL 8 (через
// testcontainers-go) и применяет все миграции, чтобы интеграционные тесты
// infrastructure/storage выполнялись на той же схеме, что и в продакшене.
// Собирается только с тегом `-tags=integration` — требует запущенный демон Docker.
package mysqlcontainer

import (
	"context"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"

	"github.com/meindokuse/task-service/migrations"
)

const (
	testDatabase = "task_service_test"
	testUser     = "test"
	testPassword = "test"
)

// Setup запускает контейнер MySQL, применяет к нему миграции и возвращает
// подключённый *sqlx.DB. Контейнер и соединение закрываются через t.Cleanup.
func Setup(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	container, err := tcmysql.Run(ctx, "mysql:8.0",
		tcmysql.WithDatabase(testDatabase),
		tcmysql.WithUsername(testUser),
		tcmysql.WithPassword(testPassword),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	dsn, err := container.ConnectionString(ctx, "parseTime=true&multiStatements=true")
	if err != nil {
		t.Fatalf("build mysql dsn: %v", err)
	}

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		t.Fatalf("connect to mysql container: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := migrateUp(db, testDatabase); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return db
}

func migrateUp(db *sqlx.DB, databaseName string) error {
	driver, err := mysqlmigrate.WithInstance(db.DB, &mysqlmigrate.Config{})
	if err != nil {
		return err
	}

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, databaseName, driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
