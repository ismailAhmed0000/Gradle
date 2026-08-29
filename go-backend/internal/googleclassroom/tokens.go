package googleclassroom

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gradle-go-backend/internal/config"
	"gradle-go-backend/internal/models"
)

var ErrNotConnected = errors.New("google account not connected")

// encryptionKey derives a 32-byte AES-256 key from whatever length secret is
// configured, so operators don't have to generate/format a key specially.
func encryptionKey(cfg *config.Config) ([]byte, error) {
	if cfg.TokenEncryptionKey == "" {
		return nil, fmt.Errorf("GOOGLE_TOKEN_ENCRYPTION_KEY is not configured")
	}
	sum := sha256.Sum256([]byte(cfg.TokenEncryptionKey))
	return sum[:], nil
}

func encrypt(cfg *config.Config, plaintext string) (string, error) {
	key, err := encryptionKey(cfg)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func decrypt(cfg *config.Config, encoded string) (string, error) {
	key, err := encryptionKey(cfg)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// SaveToken upserts the (encrypted) refresh token for a user, replacing
// whatever was there if they reconnect.
func SaveToken(db *gorm.DB, cfg *config.Config, userID uuid.UUID, refreshToken, email string, scopes []string) error {
	encrypted, err := encrypt(cfg, refreshToken)
	if err != nil {
		return err
	}
	record := models.GoogleOAuthToken{
		UserID:                userID,
		RefreshTokenEncrypted: encrypted,
		GoogleEmail:           email,
		Scope:                 strings.Join(scopes, " "),
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"refresh_token_encrypted", "google_email", "scope", "updated_at"}),
	}).Create(&record).Error
}

func IsConnected(db *gorm.DB, userID uuid.UUID) (bool, string, error) {
	var record models.GoogleOAuthToken
	err := db.Where("user_id = ?", userID).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, record.GoogleEmail, nil
}

func Disconnect(db *gorm.DB, userID uuid.UUID) error {
	return db.Where("user_id = ?", userID).Delete(&models.GoogleOAuthToken{}).Error
}

// HTTPClientFor returns an http.Client that transparently mints (and
// refreshes) access tokens from the user's stored refresh token. The access
// token itself is never persisted.
func HTTPClientFor(ctx context.Context, db *gorm.DB, cfg *config.Config, userID uuid.UUID, flow string) (*http.Client, error) {
	var record models.GoogleOAuthToken
	if err := db.Where("user_id = ?", userID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotConnected
		}
		return nil, err
	}
	refreshToken, err := decrypt(cfg, record.RefreshTokenEncrypted)
	if err != nil {
		return nil, err
	}
	oauthCfg, err := OAuthConfig(cfg, flow)
	if err != nil {
		return nil, err
	}
	tokenSource := oauthCfg.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(-time.Minute), // force an immediate refresh
	})
	return oauth2.NewClient(ctx, tokenSource), nil
}
