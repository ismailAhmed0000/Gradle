package models

import (
	"time"

	"github.com/google/uuid"
)

// GoogleOAuthToken holds one user's (teacher or student) Google refresh
// token. Only the refresh token is persisted — access tokens are minted on
// demand via oauth2.TokenSource and never stored.
type GoogleOAuthToken struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	RefreshTokenEncrypted string
	GoogleEmail           string
	Scope                 string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
