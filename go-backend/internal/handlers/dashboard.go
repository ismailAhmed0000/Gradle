package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"gradle-go-backend/internal/middleware"
	"gradle-go-backend/internal/models"
)

type DashboardHandler struct {
	DB *pgxpool.Pool
}

func NewDashboardHandler(db *pgxpool.Pool) *DashboardHandler {
	return &DashboardHandler{DB: db}
}

func (h *DashboardHandler) Summary(c *fiber.Ctx) error {
	ownerID := c.Locals(middleware.LocalsUserID)

	activity, err := h.weeklyActivity(c.Context(), ownerID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load weekly activity")
	}

	var todayPages int
	if err := h.DB.QueryRow(
		c.Context(),
		`SELECT COUNT(*) FROM submission_pages sp
		 JOIN submissions s ON s.id = sp.submission_id
		 JOIN assignments a ON a.id = s.assignment_id
		 WHERE a.owner_id = $1 AND sp.created_at::date = CURRENT_DATE`,
		ownerID,
	).Scan(&todayPages); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load today's activity")
	}

	var composited, totalSubmissions int
	if err := h.DB.QueryRow(
		c.Context(),
		`SELECT COUNT(*) FILTER (WHERE s.status = 'composited'), COUNT(*)
		 FROM submissions s JOIN assignments a ON a.id = s.assignment_id
		 WHERE a.owner_id = $1 AND s.created_at >= CURRENT_DATE - INTERVAL '6 days'`,
		ownerID,
	).Scan(&composited, &totalSubmissions); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load submission stats")
	}

	pagesThisWeek := 0
	for _, day := range activity {
		pagesThisWeek += day.PagesScanned
	}

	return c.JSON(models.DashboardSummary{
		TodayPagesScanned:    todayPages,
		WeeklyActivity:       activity,
		SubmittedThisWeek:    totalSubmissions,
		PendingThisWeek:      totalSubmissions - composited,
		SubmissionsThisWeek:  totalSubmissions,
		PagesScannedThisWeek: pagesThisWeek,
	})
}

func (h *DashboardHandler) weeklyActivity(ctx context.Context, ownerID interface{}) ([]models.DailyActivity, error) {
	rows, err := h.DB.Query(
		ctx,
		`WITH days AS (
		   SELECT generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day')::date AS day
		 ),
		 pages AS (
		   SELECT sp.created_at::date AS day, sp.id
		   FROM submission_pages sp
		   JOIN submissions s ON s.id = sp.submission_id
		   JOIN assignments a ON a.id = s.assignment_id
		   WHERE a.owner_id = $1
		 )
		 SELECT d.day, COUNT(p.id)
		 FROM days d LEFT JOIN pages p ON p.day = d.day
		 GROUP BY d.day ORDER BY d.day`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activity := []models.DailyActivity{}
	for rows.Next() {
		var day time.Time
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		activity = append(activity, models.DailyActivity{
			Date:         day.Format("2006-01-02"),
			Weekday:      weekdayLetter(day.Weekday()),
			PagesScanned: count,
		})
	}
	return activity, rows.Err()
}

func weekdayLetter(d time.Weekday) string {
	switch d {
	case time.Sunday:
		return "S"
	case time.Monday:
		return "M"
	case time.Tuesday:
		return "T"
	case time.Wednesday:
		return "W"
	case time.Thursday:
		return "T"
	case time.Friday:
		return "F"
	case time.Saturday:
		return "S"
	default:
		return ""
	}
}
