package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"

	"gradle-go-backend/internal/config"
	"gradle-go-backend/internal/db"
	"gradle-go-backend/internal/handlers"
	"gradle-go-backend/internal/queue"
	"gradle-go-backend/internal/router"
	"gradle-go-backend/internal/storage"
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

	s3, err := storage.NewS3Storage(cfg.S3EndpointURL, cfg.S3PublicURL, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3Region)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	jobQueue, err := queue.NewJobQueue(cfg.RedisURL)
	if err != nil {
		log.Fatalf("queue: %v", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: jsonErrorHandler,
	})

	h := &router.Handlers{
		Auth:        handlers.NewAuthHandler(pool, cfg),
		Assignments: handlers.NewAssignmentHandler(pool, s3),
		Submissions: handlers.NewSubmissionHandler(pool, s3, jobQueue),
		Internal:    handlers.NewInternalHandler(pool, jobQueue),
	}
	router.Setup(app, h, cfg)

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
