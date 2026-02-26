package types

import (
	"github.com/google/uuid"
)

type UserWithAuth struct {
	ID      uuid.UUID `json:"id" validate:"required"`
	Email   string    `json:"email" validate:"required,email"`
	IsVerif bool      `json:"is_verif" validate:"omitempty"`
	TeamId  string    `json:"team_id" validate:"omitempty"`
	RoleId  string    `json:"role_id" validate:"omitempty"`
}
