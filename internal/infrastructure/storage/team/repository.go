// Package team реализует интерфейс storage.Repository для команд поверх MySQL,
// включая сложный запрос (a): количество участников и количество завершённых
// за последние 7 дней задач по каждой команде — через JOIN 3 таблиц и агрегацию.
package team

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/pkg/terror"
	"github.com/meindokuse/task-service/internal/pkg/txmanager"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, t entity.Team) (uint64, error) {
	exec := txmanager.Ext(ctx, r.db)
	res, err := exec.ExecContext(ctx,
		`INSERT INTO teams (name, created_by) VALUES (?, ?)`,
		t.Name, t.CreatedBy,
	)
	if err != nil {
		return 0, terror.Internal("create team", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, terror.Internal("read last insert id", err)
	}
	return uint64(id), nil
}

func (r *Repository) GetByID(ctx context.Context, id uint64) (*entity.Team, error) {
	var dao teamDAO
	err := r.db.GetContext(ctx, &dao,
		`SELECT id, name, created_by, created_at FROM teams WHERE id = ?`, id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, terror.NotFound("team not found")
	}
	if err != nil {
		return nil, terror.Internal("get team by id", err)
	}

	team := dao.toEntity()
	return &team, nil
}

// ListForUser возвращает все команды, в которых состоит указанный пользователь.
func (r *Repository) ListForUser(ctx context.Context, userID uint64) ([]entity.Team, error) {
	var daos []teamDAO
	err := r.db.SelectContext(ctx, &daos, `
		SELECT t.id, t.name, t.created_by, t.created_at
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ?
		ORDER BY t.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, terror.Internal("list teams for user", err)
	}

	teams := make([]entity.Team, 0, len(daos))
	for _, d := range daos {
		teams = append(teams, d.toEntity())
	}
	return teams, nil
}

// Stats — это сложный запрос (a): для каждой команды название + количество
// участников + количество задач, переведённых в статус "done" за последние 7 дней.
// JOIN 3 таблиц + агрегация.
func (r *Repository) Stats(ctx context.Context) ([]entity.TeamStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			t.id,
			t.name,
			COUNT(DISTINCT tm.user_id) AS member_count,
			COUNT(DISTINCT CASE
				WHEN tk.status = 'done' AND tk.updated_at >= NOW() - INTERVAL 7 DAY
				THEN tk.id
			END) AS done_last_7_days
		FROM teams t
		LEFT JOIN team_members tm ON tm.team_id = t.id
		LEFT JOIN tasks tk ON tk.team_id = t.id
		GROUP BY t.id, t.name
		ORDER BY t.id`,
	)
	if err != nil {
		return nil, terror.Internal("team stats", err)
	}
	defer rows.Close()

	var stats []entity.TeamStats
	for rows.Next() {
		var s entity.TeamStats
		if err := rows.Scan(&s.TeamID, &s.Name, &s.MemberCount, &s.DoneLast7Days); err != nil {
			return nil, terror.Internal("scan team stats", err)
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, terror.Internal("iterate team stats", err)
	}

	return stats, nil
}
