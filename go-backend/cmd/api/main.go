package main

import (
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

	gormDB, err := db.NewGormDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer sqlDB.Close()

	s3, err := storage.NewS3Storage(cfg.S3EndpointURL, cfg.S3PublicURL, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3Region, cfg.S3VirtualHost)
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
		Auth:        handlers.NewAuthHandler(gormDB, cfg),
		Assignments: handlers.NewAssignmentHandler(gormDB, s3),
		Submissions: handlers.NewSubmissionHandler(gormDB, s3, jobQueue),
		Internal:    handlers.NewInternalHandler(gormDB, jobQueue),
		Dashboard:   handlers.NewDashboardHandler(gormDB),
		Subjects:    handlers.NewSubjectHandler(gormDB),
		Students:    handlers.NewStudentHandler(gormDB),
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
