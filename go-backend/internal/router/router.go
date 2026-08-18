package router

import (
	"github.com/gofiber/fiber/v2"

	"gradle-go-backend/internal/config"
	"gradle-go-backend/internal/handlers"
	"gradle-go-backend/internal/middleware"
)

func Setup(app *fiber.App, authHandler *handlers.AuthHandler, cfg *config.Config) {
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	api := app.Group("/api")

	authGroup := api.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Get("/me", middleware.RequireAuth(cfg.JWTSecret), authHandler.Me)
}
