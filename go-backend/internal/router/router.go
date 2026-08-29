package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

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
	Subjects    *handlers.SubjectHandler
	Students    *handlers.StudentHandler
	Google      *handlers.GoogleIntegrationHandler
}

func Setup(app *fiber.App, h *Handlers, cfg *config.Config) {
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PATCH, DELETE, OPTIONS",
	}))

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
	assignmentsGroup.Post("/", h.Assignments.Create)
	assignmentsGroup.Get("/:id", h.Assignments.GetByID)
	assignmentsGroup.Post("/:id/files", h.Assignments.UploadFile)
	assignmentsGroup.Get("/:id/submissions", h.Submissions.ListForAssignment)
	assignmentsGroup.Post("/:id/submissions", h.Submissions.Create)

	submissionsGroup := api.Group("/submissions", requireAuth)
	submissionsGroup.Get("/:id", h.Submissions.Get)
	submissionsGroup.Post("/:id/pages", h.Submissions.UploadPage)
	submissionsGroup.Get("/:id/composited", h.Submissions.GetComposited)
	submissionsGroup.Patch("/:id/grade", h.Submissions.Grade)

	subjectsGroup := api.Group("/subjects", requireAuth)
	subjectsGroup.Get("/", h.Subjects.List)
	subjectsGroup.Post("/", h.Subjects.Create)
	subjectsGroup.Get("/:id", h.Subjects.GetByID)

	googleGroup := api.Group("/integrations/google")
	googleGroup.Get("/callback", h.Google.Callback)
	googleGroup.Get("/student/auth-url", h.Google.StudentAuthURL)
	googleGroup.Get("/teacher/auth-url", requireAuth, h.Google.TeacherAuthURL)
	googleGroup.Get("/status", requireAuth, h.Google.Status)
	googleGroup.Delete("/", requireAuth, h.Google.Disconnect)
	googleGroup.Get("/courses", requireAuth, h.Google.Courses)
	googleGroup.Get("/courses/:id/coursework", requireAuth, h.Google.CourseWork)
	googleGroup.Post("/import", requireAuth, h.Google.Import)

	requireAdmin := middleware.RequireAdmin()

	studentsGroup := api.Group("/students", requireAuth)
	studentsGroup.Get("/", h.Students.List)
	studentsGroup.Post("/", requireAdmin, h.Students.Create)
	studentsGroup.Get("/:id", h.Students.Get)
	studentsGroup.Post("/:id/enroll", requireAdmin, h.Students.Enroll)

	requireInternalToken := middleware.RequireInternalToken(cfg.InternalAPIToken)

	internalGroup := app.Group("/internal", requireInternalToken)
	internalGroup.Get("/answer-regions/:id/context", h.Internal.AnswerRegionContext)
	internalGroup.Patch("/answer-regions/:id/start", h.Internal.StartAnswerRegion)
	internalGroup.Patch("/answer-regions/:id", h.Internal.ReportAnswerRegionResult)
	internalGroup.Get("/composited-documents/:id/context", h.Internal.CompositeContext)
	internalGroup.Patch("/composited-documents/:id/start", h.Internal.StartComposite)
	internalGroup.Patch("/composited-documents/:id", h.Internal.ReportCompositeResult)
}
