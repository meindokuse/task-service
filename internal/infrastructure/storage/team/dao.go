package team

import (
	"time"

	"github.com/meindokuse/task-service/internal/domain/entity"
)

type teamDAO struct {
	ID        uint64    `db:"id"`
	Name      string    `db:"name"`
	CreatedBy uint64    `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
}

func (d teamDAO) toEntity() entity.Team {
	return entity.Team{
		ID:        d.ID,
		Name:      d.Name,
		CreatedBy: d.CreatedBy,
		CreatedAt: d.CreatedAt,
	}
}
