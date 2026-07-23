// Package task реализует интерфейс storage.Repository для задач, включая
// сложные запросы (b) топ авторов задач по командам (CTE + оконная функция) и
// (c) задачи, чей исполнитель не является участником команды (проверка целостности).
package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

func (r *Repository) Create(ctx context.Context, t entity.Task) (uint64, error) {
	var assigneeID sql.NullInt64
	if t.AssigneeID != nil {
		assigneeID = sql.NullInt64{Int64: int64(*t.AssigneeID), Valid: true}
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (team_id, title, description, status, assignee_id, created_by)
		VALUES (?, ?, ?, ?, ?, ?)`,
		t.TeamID, t.Title, t.Description, string(t.Status), assigneeID, t.CreatedBy,
	)
	if err != nil {
		return 0, terror.Internal("create task", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, terror.Internal("read last insert id", err)
	}
	return uint64(id), nil
}

func (r *Repository) GetByID(ctx context.Context, id uint64) (*entity.Task, error) {
	var dao taskDAO
	err := r.db.GetContext(ctx, &dao, `
		SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at
		FROM tasks WHERE id = ?`, id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, terror.NotFound("task not found")
	}
	if err != nil {
		return nil, terror.Internal("get task by id", err)
	}

	t := dao.toEntity()
	return &t, nil
}

// List применяет фильтрацию и пагинацию на уровне БД, а также возвращает
// общее количество строк (без учёта пагинации), чтобы вызывающий код мог
// сформировать метаданные страницы.
func (r *Repository) List(ctx context.Context, f entity.TaskFilter) (entity.TaskList, error) {
	where := []string{"team_id = ?"}
	args := []interface{}{f.TeamID}

	if f.Status != nil {
		where = append(where, "status = ?")
		args = append(args, string(*f.Status))
	}
	if f.AssigneeID != nil {
		where = append(where, "assignee_id = ?")
		args = append(args, *f.AssigneeID)
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM tasks WHERE %s`, whereClause)
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return entity.TaskList{}, terror.Internal("count tasks", err)
	}

	page, pageSize := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	listQuery := fmt.Sprintf(`
		SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at
		FROM tasks
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT ? OFFSET ?`, whereClause)
	listArgs := append(append([]interface{}{}, args...), pageSize, offset)

	var daos []taskDAO
	if err := r.db.SelectContext(ctx, &daos, listQuery, listArgs...); err != nil {
		return entity.TaskList{}, terror.Internal("list tasks", err)
	}

	items := make([]entity.Task, 0, len(daos))
	for _, d := range daos {
		items = append(items, d.toEntity())
	}

	return entity.TaskList{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Update перезаписывает изменяемые поля задачи (title, description, status,
// assignee). Сравнение с предыдущими значениями для истории происходит в
// use case, до вызова этого метода.
func (r *Repository) Update(ctx context.Context, t entity.Task) error {
	var assigneeID sql.NullInt64
	if t.AssigneeID != nil {
		assigneeID = sql.NullInt64{Int64: int64(*t.AssigneeID), Valid: true}
	}

	exec := txmanager.Ext(ctx, r.db)
	res, err := exec.ExecContext(ctx, `
		UPDATE tasks SET title = ?, description = ?, status = ?, assignee_id = ?
		WHERE id = ?`,
		t.Title, t.Description, string(t.Status), assigneeID, t.ID,
	)
	if err != nil {
		return terror.Internal("update task", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return terror.Internal("read rows affected", err)
	}
	if affected == 0 {
		return terror.NotFound("task not found")
	}
	return nil
}

// TopCreators — это сложный запрос (b): топ-3 пользователя по количеству
// созданных за текущий месяц задач, ранжируемые по каждой команде через
// CTE + оконную функцию ROW_NUMBER(), отфильтрованные по запрошенной команде.
func (r *Repository) TopCreators(ctx context.Context, teamID uint64) ([]entity.TopCreator, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH ranked AS (
			SELECT
				tk.team_id,
				tk.created_by AS user_id,
				u.name AS user_name,
				COUNT(*) AS tasks_count,
				ROW_NUMBER() OVER (PARTITION BY tk.team_id ORDER BY COUNT(*) DESC) AS rnk
			FROM tasks tk
			JOIN users u ON u.id = tk.created_by
			WHERE tk.created_at >= DATE_FORMAT(CURDATE(), '%Y-%m-01')
			GROUP BY tk.team_id, tk.created_by, u.name
		)
		SELECT team_id, user_id, user_name, tasks_count, rnk
		FROM ranked
		WHERE rnk <= 3 AND team_id = ?
		ORDER BY rnk`,
		teamID,
	)
	if err != nil {
		return nil, terror.Internal("top creators", err)
	}
	defer rows.Close()

	var result []entity.TopCreator
	for rows.Next() {
		var c entity.TopCreator
		if err := rows.Scan(&c.TeamID, &c.UserID, &c.UserName, &c.TasksCount, &c.RankInTeam); err != nil {
			return nil, terror.Internal("scan top creators", err)
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, terror.Internal("iterate top creators", err)
	}

	return result, nil
}

// OrphanedAssignees — это сложный запрос (c): задачи в указанной команде, чей
// исполнитель не является (или перестал являться) участником этой команды —
// проверка целостности данных, выраженная через коррелированный подзапрос NOT EXISTS.
func (r *Repository) OrphanedAssignees(ctx context.Context, teamID uint64) ([]entity.Task, error) {
	var daos []taskDAO
	err := r.db.SelectContext(ctx, &daos, `
		SELECT tk.id, tk.team_id, tk.title, tk.description, tk.status, tk.assignee_id, tk.created_by, tk.created_at, tk.updated_at
		FROM tasks tk
		WHERE tk.team_id = ?
		  AND tk.assignee_id IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM team_members tm
			WHERE tm.team_id = tk.team_id AND tm.user_id = tk.assignee_id
		  )
		ORDER BY tk.id`,
		teamID,
	)
	if err != nil {
		return nil, terror.Internal("orphaned assignees", err)
	}

	items := make([]entity.Task, 0, len(daos))
	for _, d := range daos {
		items = append(items, d.toEntity())
	}
	return items, nil
}
