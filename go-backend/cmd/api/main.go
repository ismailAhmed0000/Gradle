package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"

	"gradle-go-backend/internal/config"
	"gradle-go-backend/internal/db"
	"gradle-go-backend/internal/handlers"
	"gradle-go-backend/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pool, err := db.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	app := fiber.New(fiber.Config{
		ErrorHandler: jsonErrorHandler,
	})

	authHandler := handlers.NewAuthHandler(pool, cfg)
	router.Setup(app, authHandler, cfg)

	log.Printf("listening on :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func jsonErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "internal server error"

	var fiberErr *fiber.Error
	if e, ok := err.(*fiber.Error); ok {
		fiberErr = e
		code = fiberErr.Code
		message = fiberErr.Message
	}

	return c.Status(code).JSON(fiber.Map{"error": message})
}
