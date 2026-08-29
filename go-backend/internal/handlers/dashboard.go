package handlers

import (
	"errors"
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

	var totalStudents, totalSubjects int64
	if err := h.DB.Model(&models.Student{}).Where("owner_id = ?", ownerID).Count(&totalStudents).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count students")
	}
	if err := h.DB.Model(&models.Subject{}).Where("owner_id = ?", ownerID).Count(&totalSubjects).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count subjects")
	}

	pendingSubmissions, pendingToGrade, err := h.assignmentPendingCounts(ownerID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count pending assignments")
	}

	return c.JSON(models.DashboardSummary{
		TodayPagesScanned:    int(todayPages),
		WeeklyActivity:       activity,
		SubmittedThisWeek:    int(totalSubmissions),
		PendingThisWeek:      int(totalSubmissions - composited),
		SubmissionsThisWeek:  int(totalSubmissions),
		PagesScannedThisWeek: pagesThisWeek,
		TotalStudents:        int(totalStudents),
		TotalSubjects:        int(totalSubjects),
		PendingSubmissions:   pendingSubmissions,
		PendingToGrade:       pendingToGrade,
	})
}

func (h *DashboardHandler) assignmentPendingCounts(ownerID uuid.UUID) (pendingSubmissions, pendingToGrade int, err error) {
	var assignments []models.Assignment
	if err = h.DB.Select("id", "due_date").Where("owner_id = ?", ownerID).Find(&assignments).Error; err != nil {
		return 0, 0, err
	}

	for _, a := range assignments {
		var latest models.Submission
		lerr := h.DB.Where("assignment_id = ?", a.ID).Order("created_at DESC").First(&latest).Error
		var latestStatus *string
		switch {
		case lerr == nil:
			s := string(latest.Status)
			latestStatus = &s
		case errors.Is(lerr, gorm.ErrRecordNotFound):
		default:
			return 0, 0, lerr
		}

		switch computeAssignmentStatus(a.DueDate, latestStatus) {
		case models.AssignmentStatusPending:
			pendingSubmissions++
		case models.AssignmentStatusSubmitted:
			pendingToGrade++
		}
	}

	return pendingSubmissions, pendingToGrade, nil
}

type dailyActivityRow struct {
	Day                time.Time
	PagesScanned       int
	PendingSubmissions int
	PendingToGrade     int
}

// weeklyActivity builds a full 7-day calendar (including days with zero
// scans) and counts, per day: pages scanned, and — of the submissions
// created that day — how many are still sitting in 'pending' (student
// hasn't finished scanning) or 'processing' (scanned, not yet composited)
// as of right now. That's a backlog-by-creation-day view, not a live status
// snapshot, so a submission created 3 days ago that got composited today no
// longer counts on day 3. A generate_series + multiple left joins is
// clearer as one SQL statement than as a chain of ORM calls, so this stays
// a Raw query executed through GORM rather than the query builder.
func (h *DashboardHandler) weeklyActivity(ownerID uuid.UUID) ([]models.DailyActivity, error) {
	var rows []dailyActivityRow
	err := h.DB.Raw(
		`WITH days AS (
		   SELECT generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day')::date AS day
		 ),
		 pages_per_day AS (
		   SELECT sp.created_at::date AS day, COUNT(*) AS pages_scanned
		   FROM submission_pages sp
		   JOIN submissions s ON s.id = sp.submission_id
		   JOIN assignments a ON a.id = s.assignment_id
		   WHERE a.owner_id = ?
		   GROUP BY sp.created_at::date
		 ),
		 submissions_per_day AS (
		   SELECT s.created_at::date AS day,
		          COUNT(*) FILTER (WHERE s.status = 'pending') AS pending_submissions,
		          COUNT(*) FILTER (WHERE s.status = 'processing') AS pending_to_grade
		   FROM submissions s
		   JOIN assignments a ON a.id = s.assignment_id
		   WHERE a.owner_id = ?
		   GROUP BY s.created_at::date
		 )
		 SELECT d.day AS day,
		        COALESCE(pp.pages_scanned, 0) AS pages_scanned,
		        COALESCE(sp.pending_submissions, 0) AS pending_submissions,
		        COALESCE(sp.pending_to_grade, 0) AS pending_to_grade
		 FROM days d
		 LEFT JOIN pages_per_day pp ON pp.day = d.day
		 LEFT JOIN submissions_per_day sp ON sp.day = d.day
		 ORDER BY d.day`,
		ownerID, ownerID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	activity := make([]models.DailyActivity, len(rows))
	for i, row := range rows {
		activity[i] = models.DailyActivity{
			Date:               row.Day.Format("2006-01-02"),
			Weekday:            weekdayLetter(row.Day.Weekday()),
			PagesScanned:       row.PagesScanned,
			PendingSubmissions: row.PendingSubmissions,
			PendingToGrade:     row.PendingToGrade,
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
