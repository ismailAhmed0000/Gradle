package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"gradle-go-backend/internal/middleware"
	"gradle-go-backend/internal/models"
)

type DashboardHandler struct {
	DB *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{DB: db}
}

func (h *DashboardHandler) Summary(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	activity, err := h.weeklyActivity(ownerID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load weekly activity")
	}

	var todayPages int64
	if err := h.DB.Table("submission_pages").
		Joins("JOIN submissions ON submissions.id = submission_pages.submission_id").
		Joins("JOIN assignments ON assignments.id = submissions.assignment_id").
		Where("assignments.owner_id = ? AND submission_pages.created_at::date = CURRENT_DATE", ownerID).
		Count(&todayPages).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load today's activity")
	}

	weekStart := time.Now().AddDate(0, 0, -6)

	var totalSubmissions int64
	if err := h.DB.Table("submissions").
		Joins("JOIN assignments ON assignments.id = submissions.assignment_id").
		Where("assignments.owner_id = ? AND submissions.created_at >= ?", ownerID, weekStart).
		Count(&totalSubmissions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load submission stats")
	}

	var composited int64
	if err := h.DB.Table("submissions").
		Joins("JOIN assignments ON assignments.id = submissions.assignment_id").
		Where("assignments.owner_id = ? AND submissions.created_at >= ? AND submissions.status = ?",
			ownerID, weekStart, models.SubmissionStatusComposited).
		Count(&composited).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load submission stats")
	}

	pagesThisWeek := 0
	for _, day := range activity {
		pagesThisWeek += day.PagesScanned
	}

	return c.JSON(models.DashboardSummary{
		TodayPagesScanned:    int(todayPages),
		WeeklyActivity:       activity,
		SubmittedThisWeek:    int(totalSubmissions),
		PendingThisWeek:      int(totalSubmissions - composited),
		SubmissionsThisWeek:  int(totalSubmissions),
		PagesScannedThisWeek: pagesThisWeek,
	})
}

type dailyActivityRow struct {
	Day          time.Time
	PagesScanned int
}

// weeklyActivity builds a full 7-day calendar (including days with zero
// scans) and counts pages scanned per day — a generate_series + left join
// that's clearer as one SQL statement than as a chain of ORM calls, so it
// stays a Raw query executed through GORM rather than the query builder.
func (h *DashboardHandler) weeklyActivity(ownerID uuid.UUID) ([]models.DailyActivity, error) {
	var rows []dailyActivityRow
	err := h.DB.Raw(
		`WITH days AS (
		   SELECT generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day')::date AS day
		 ),
		 pages AS (
		   SELECT sp.created_at::date AS day, sp.id
		   FROM submission_pages sp
		   JOIN submissions s ON s.id = sp.submission_id
		   JOIN assignments a ON a.id = s.assignment_id
		   WHERE a.owner_id = ?
		 )
		 SELECT d.day AS day, COUNT(p.id) AS pages_scanned
		 FROM days d LEFT JOIN pages p ON p.day = d.day
		 GROUP BY d.day ORDER BY d.day`,
		ownerID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	activity := make([]models.DailyActivity, len(rows))
	for i, row := range rows {
		activity[i] = models.DailyActivity{
			Date:         row.Day.Format("2006-01-02"),
			Weekday:      weekdayLetter(row.Day.Weekday()),
			PagesScanned: row.PagesScanned,
		}
	}
	return activity, nil
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
