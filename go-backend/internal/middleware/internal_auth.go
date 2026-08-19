package middleware

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"
)

func RequireInternalToken(token string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provided := c.Get("X-Internal-Token")
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid internal token")
		}
		return c.Next()
	}
}
