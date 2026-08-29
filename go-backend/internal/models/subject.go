package models

import (
	"time"

	"github.com/google/uuid"
)

type Subject struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID   uuid.UUID `json:"owner_id" gorm:"type:uuid"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type SubjectDetail struct {
	Subject
	Students []Student `json:"students"`
}
