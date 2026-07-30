// Package txmanager реализует передачу транзакции MySQL через context: репозитории
// достают активный *sqlx.Tx (если он есть) из ctx через Ext, благодаря чему use case
// может обернуть несколько вызовов репозиториев в одну атомарную операцию,
// а сами репозитории при этом ничего не знают друг о друге.
package txmanager

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type ctxKey struct{}

type Manager struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Manager {
	return &Manager{db: db}
}

// WithinTx открывает транзакцию, выполняет fn с прикреплённой к ctx транзакцией
// и делает commit при успехе или rollback при ошибке/панике.
//
// Не реентерабелен: всегда открывает новую транзакцию, не проверяя, есть ли
// уже активная в ctx. Вложенный вызов WithinTx внутри другого WithinTx
// откроет вторую независимую транзакцию на отдельном соединении, что может
// привести к самоблокировке при захвате той же строки (например, FOR UPDATE).
// Ни один текущий вызывающий код так не делает.
func (m *Manager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	err = fn(context.WithValue(ctx, ctxKey{}, tx))
	return err
}

// Ext возвращает активную транзакцию из ctx или fallback, если её нет —
// репозитории вызывают эту функцию, чтобы стать "транзакционно-осведомлёнными"
// без явного параметра tx.
func Ext(ctx context.Context, fallback *sqlx.DB) sqlx.ExtContext {
	if tx, ok := ctx.Value(ctxKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return fallback
}
