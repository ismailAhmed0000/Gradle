package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gradle-go-backend/internal/middleware"
	"gradle-go-backend/internal/models"
)

type StudentHandler struct {
	DB *gorm.DB
}

func NewStudentHandler(db *gorm.DB) *StudentHandler {
	return &StudentHandler{DB: db}
}

// List returns every student for an admin, or — for a teacher — only
// students enrolled in a subject that teacher owns.
func (h *StudentHandler) List(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	query := h.DB.Model(&models.Student{})
	if !middleware.IsAdmin(c) {
		query = query.
			Select("DISTINCT students.*").
			Joins("JOIN enrollments ON enrollments.student_id = students.id").
			Joins("JOIN subjects ON subjects.id = enrollments.subject_id").
			Where("subjects.owner_id = ?", userID)
	}

	students := []models.Student{}
	if err := query.Order("students.name ASC").Find(&students).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list students")
	}

	scopeOwnerID := teacherScope(c, userID)
	summaries := make([]models.StudentSummary, len(students))
	for i, s := range students {
		subjects, err := h.listSubjectsForStudent(s.ID, scopeOwnerID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to load enrolled subjects")
		}
		summaries[i] = models.StudentSummary{Student: s, Subjects: subjects}
	}

	return c.JSON(summaries)
}

// teacherScope returns nil for an admin (no ownership restriction) or the
// requesting user's id for a teacher (restricted to their own subjects).
func teacherScope(c *fiber.Ctx, userID uuid.UUID) *uuid.UUID {
	if middleware.IsAdmin(c) {
		return nil
	}
	return &userID
}

type createStudentRequest struct {
	Name  string  `json:"name"`
	Email *string `json:"email"`
}

// Create is admin-only (enforced by RequireAdmin in the router).
func (h *StudentHandler) Create(c *fiber.Ctx) error {
	adminID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	var req createStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	student := models.Student{OwnerID: adminID, Name: req.Name, Email: req.Email}
	if err := h.DB.Create(&student).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return fiber.NewError(fiber.StatusConflict, "a student with this name already exists")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create student")
	}

	return c.Status(fiber.StatusCreated).JSON(student)
}

type enrollRequest struct {
	SubjectID uuid.UUID `json:"subject_id"`
}

// Enroll is admin-only (enforced by RequireAdmin in the router), so it isn't
// scoped to any one teacher's subjects — an admin can enroll any student
// into any subject.
func (h *StudentHandler) Enroll(c *fiber.Ctx) error {
	studentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid student id")
	}

	var req enrollRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.SubjectID == uuid.Nil {
		return fiber.NewError(fiber.StatusBadRequest, "subject_id is required")
	}

	var student models.Student
	if err := h.DB.Select("id").First(&student, "id = ?", studentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "student not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up student")
	}

	var subject models.Subject
	if err := h.DB.Select("id").First(&subject, "id = ?", req.SubjectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "subject not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up subject")
	}

	enrollment := models.Enrollment{StudentID: studentID, SubjectID: req.SubjectID}
	if err := h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "student_id"}, {Name: "subject_id"}},
		DoNothing: true,
	}).Create(&enrollment).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to enroll student")
	}

	subjects, err := h.listSubjectsForStudent(studentID, nil)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load enrolled subjects")
	}

	return c.Status(fiber.StatusCreated).JSON(subjects)
}

// Get returns a student's enrolled subjects and work. An admin sees
// everything; a teacher sees only the slice that runs through their own
// subjects, and gets a 404 (not a 403) if the student has no relationship
// to any subject they own — same as if the student didn't exist for them.
func (h *StudentHandler) Get(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	studentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid student id")
	}

	var student models.Student
	if err := h.DB.First(&student, "id = ?", studentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "student not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up student")
	}

	scopeOwnerID := teacherScope(c, userID)

	subjects, err := h.listSubjectsForStudent(studentID, scopeOwnerID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load enrolled subjects")
	}
	if scopeOwnerID != nil && len(subjects) == 0 {
		return fiber.NewError(fiber.StatusNotFound, "student not found")
	}

	detail := models.StudentDetail{Student: student, Subjects: subjects}

	detail.Submissions, err = h.listWorkForStudent(studentID, scopeOwnerID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load student work")
	}

	return c.JSON(detail)
}

func (h *StudentHandler) listSubjectsForStudent(studentID uuid.UUID, ownerID *uuid.UUID) ([]models.Subject, error) {
	query := h.DB.Joins("JOIN enrollments ON enrollments.subject_id = subjects.id").
		Where("enrollments.student_id = ?", studentID)
	if ownerID != nil {
		query = query.Where("subjects.owner_id = ?", *ownerID)
	}

	subjects := []models.Subject{}
	err := query.Order("subjects.name ASC").Find(&subjects).Error
	return subjects, err
}

// listWorkForStudent returns every assignment in a subject the student is
// enrolled in, alongside the student's own submission for it (if they've
// started one) — so assignments they haven't touched yet still show up as
// outstanding work. When ownerID is set, both the subjects considered and
// the assignments returned are restricted to that teacher's own subjects.
func (h *StudentHandler) listWorkForStudent(studentID uuid.UUID, ownerID *uuid.UUID) ([]models.StudentSubmission, error) {
	subjectsQuery := h.DB.Table("enrollments").
		Joins("JOIN subjects ON subjects.id = enrollments.subject_id").
		Where("enrollments.student_id = ?", studentID)
	if ownerID != nil {
		subjectsQuery = subjectsQuery.Where("subjects.owner_id = ?", *ownerID)
	}

	var subjectIDs []uuid.UUID
	if err := subjectsQuery.Pluck("enrollments.subject_id", &subjectIDs).Error; err != nil {
		return nil, err
	}

	work := []models.StudentSubmission{}
	if len(subjectIDs) == 0 {
		return work, nil
	}

	var assignments []models.Assignment
	if err := h.DB.Preload("Subject").
		Where("subject_id IN ?", subjectIDs).
		Order("created_at DESC").
		Find(&assignments).Error; err != nil {
		return nil, err
	}

	for _, a := range assignments {
		w := models.StudentSubmission{AssignmentID: a.ID, AssignmentTitle: a.Title}
		if a.Subject != nil {
			w.SubjectName = &a.Subject.Name
		}

		var sub models.Submission
		err := h.DB.Where("assignment_id = ? AND student_id = ?", a.ID, studentID).First(&sub).Error
		switch {
		case err == nil:
			w.SubmissionID = &sub.ID
			w.Status = &sub.Status
			w.CreatedAt = &sub.CreatedAt
			pageCount, regionsDone, regionsTotal, countErr := submissionCounts(h.DB, sub.ID)
			if countErr != nil {
				return nil, countErr
			}
			w.PageCount = pageCount
			w.AnswerRegionsDone = regionsDone
			w.AnswerRegionsTotal = regionsTotal
		case errors.Is(err, gorm.ErrRecordNotFound):
		default:
			return nil, err
		}

		work = append(work, w)
	}

	return work, nil
}
