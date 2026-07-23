package team_member

import (
	"time"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
)

type teamMemberDAO struct {
	TeamID   uint64    `db:"team_id"`
	UserID   uint64    `db:"user_id"`
	Role     string    `db:"role"`
	JoinedAt time.Time `db:"joined_at"`
}

func (d teamMemberDAO) toEntity() entity.TeamMember {
	return entity.TeamMember{
		TeamID:   d.TeamID,
		UserID:   d.UserID,
		Role:     valueobject.Role(d.Role),
		JoinedAt: d.JoinedAt,
	}
}
