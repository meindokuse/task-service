// Package team_member реализует интерфейс storage.Repository для членства в
// командах, используемый use case'ами команд и задач для проверок доступа (RBAC).
package team_member

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
	"github.com/meindokuse/task-service/internal/pkg/txmanager"
)

const mysqlErrDuplicateEntry = 1062

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Add(ctx context.Context, teamID, userID uint64, role valueobject.Role) error {
	exec := txmanager.Ext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		teamID, userID, string(role),
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDuplicateEntry {
			return terror.Conflict("user is already a team member")
		}
		return terror.Internal("add team member", err)
	}
	return nil
}

// GetRole возвращает роль вызывающего пользователя в команде либо
// terror.NotFound, если он не участник — этот метод используется как для
// чтения членства, так и для проверки доступа к RBAC-защищённым действиям.
func (r *Repository) GetRole(ctx context.Context, teamID, userID uint64) (valueobject.Role, error) {
	var role string
	err := r.db.GetContext(ctx, &role,
		`SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID, userID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return "", terror.NotFound("not a team member")
	}
	if err != nil {
		return "", terror.Internal("get team member role", err)
	}
	return valueobject.Role(role), nil
}

func (r *Repository) IsMember(ctx context.Context, teamID, userID uint64) (bool, error) {
	_, err := r.GetRole(ctx, teamID, userID)
	if err != nil {
		var terr *terror.Error
		if errors.As(err, &terr) && terr.Kind == terror.KindNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) ListMembers(ctx context.Context, teamID uint64) ([]entity.TeamMember, error) {
	var daos []teamMemberDAO
	err := r.db.SelectContext(ctx, &daos,
		`SELECT team_id, user_id, role, joined_at FROM team_members WHERE team_id = ?`,
		teamID,
	)
	if err != nil {
		return nil, terror.Internal("list team members", err)
	}

	members := make([]entity.TeamMember, 0, len(daos))
	for _, d := range daos {
		members = append(members, d.toEntity())
	}
	return members, nil
}
