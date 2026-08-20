package router

import (
	"github.com/gofiber/fiber/v2"

	"gradle-go-backend/internal/config"
	"gradle-go-backend/internal/handlers"
	"gradle-go-backend/internal/middleware"
)

type Handlers struct {
	Auth        *handlers.AuthHandler
	Assignments *handlers.AssignmentHandler
	Submissions *handlers.SubmissionHandler
	Internal    *handlers.InternalHandler
	Dashboard   *handlers.DashboardHandler
}

func Setup(app *fiber.App, h *Handlers, cfg *config.Config) {
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	api := app.Group("/api")

	authGroup := api.Group("/auth")
	authGroup.Post("/register", h.Auth.Register)
	authGroup.Post("/login", h.Auth.Login)
	authGroup.Get("/me", middleware.RequireAuth(cfg.JWTSecret), h.Auth.Me)

	requireAuth := middleware.RequireAuth(cfg.JWTSecret)

	api.Get("/dashboard", requireAuth, h.Dashboard.Summary)

	assignmentsGroup := api.Group("/assignments", requireAuth)
	assignmentsGroup.Get("/", h.Assignments.List)
	assignmentsGroup.Get("/:id", h.Assignments.GetByID)
	assignmentsGroup.Post("/:id/submissions", h.Submissions.Create)

	submissionsGroup := api.Group("/submissions", requireAuth)
	submissionsGroup.Get("/:id", h.Submissions.Get)
	submissionsGroup.Post("/:id/pages", h.Submissions.UploadPage)
	submissionsGroup.Get("/:id/composited", h.Submissions.GetComposited)

	requireInternalToken := middleware.RequireInternalToken(cfg.InternalAPIToken)

	internalGroup := app.Group("/internal", requireInternalToken)
	internalGroup.Get("/answer-regions/:id/context", h.Internal.AnswerRegionContext)
	internalGroup.Patch("/answer-regions/:id/start", h.Internal.StartAnswerRegion)
	internalGroup.Patch("/answer-regions/:id", h.Internal.ReportAnswerRegionResult)
	internalGroup.Get("/composited-documents/:id/context", h.Internal.CompositeContext)
	internalGroup.Patch("/composited-documents/:id/start", h.Internal.StartComposite)
	internalGroup.Patch("/composited-documents/:id", h.Internal.ReportCompositeResult)
}
