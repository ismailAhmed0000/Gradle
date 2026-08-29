package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"gradle-go-backend/internal/auth"
)

const (
	LocalsUserID = "user_id"
	LocalsRole   = "role"
)

// UserIDFromContext parses the JWT subject stashed in c.Locals by
// RequireAuth back into a uuid.UUID for use as a typed GORM query param.
func UserIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, _ := c.Locals(LocalsUserID).(string)
	return uuid.Parse(raw)
}

func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return fiber.NewError(fiber.StatusUnauthorized, "missing or malformed authorization header")
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ParseToken(tokenString, jwtSecret)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals(LocalsUserID, claims.UserID)
		c.Locals(LocalsRole, claims.Role)
		return c.Next()
	}
}
