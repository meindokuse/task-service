package user

import (
	"time"

	"github.com/meindokuse/task-service/internal/domain/entity"
)

type userDAO struct {
	ID           uint64    `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Name         string    `db:"name"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func (d userDAO) toEntity() entity.User {
	return entity.User{
		ID:           d.ID,
		Email:        d.Email,
		PasswordHash: d.PasswordHash,
		Name:         d.Name,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}
