package entity

import (
	"time"

	"github.com/meindokuse/task-service/internal/domain/valueobject"
)

type Team struct {
	ID        uint64
	Name      string
	CreatedBy uint64
	CreatedAt time.Time
}

type TeamMember struct {
	TeamID   uint64
	UserID   uint64
	Role     valueobject.Role
	JoinedAt time.Time
}

// TeamStats — строка результата сложного запроса (a) "статистика по командам".
type TeamStats struct {
	TeamID        uint64
	Name          string
	MemberCount   int
	DoneLast7Days int
}

// TopCreator — строка результата сложного запроса (b) "топ-3 автора задач в команде".
type TopCreator struct {
	TeamID     uint64
	UserID     uint64
	UserName   string
	TasksCount int
	RankInTeam int
}
