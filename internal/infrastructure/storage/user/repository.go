// Package user реализует интерфейс storage.Repository для пользователей
// (объявленный в use case'ах auth/register и auth/login) поверх MySQL через sqlx.
package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

const mysqlErrDuplicateEntry = 1062

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, u entity.User) (uint64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`,
		u.Email, u.PasswordHash, u.Name,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDuplicateEntry {
			return 0, terror.Conflict("email already registered")
		}
		return 0, terror.Internal("create user", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, terror.Internal("read last insert id", err)
	}
	return uint64(id), nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	var dao userDAO
	err := r.db.GetContext(ctx, &dao,
		`SELECT id, email, password_hash, name, created_at, updated_at FROM users WHERE email = ?`,
		email,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, terror.NotFound("user not found")
	}
	if err != nil {
		return nil, terror.Internal("get user by email", err)
	}

	u := dao.toEntity()
	return &u, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint64) (*entity.User, error) {
	var dao userDAO
	err := r.db.GetContext(ctx, &dao,
		`SELECT id, email, password_hash, name, created_at, updated_at FROM users WHERE id = ?`,
		id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, terror.NotFound("user not found")
	}
	if err != nil {
		return nil, terror.Internal("get user by id", err)
	}

	u := dao.toEntity()
	return &u, nil
}
