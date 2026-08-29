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

func (h *StudentHandler) List(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	students := []models.Student{}
	if err := h.DB.Where("owner_id = ?", ownerID).Order("name ASC").Find(&students).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list students")
	}

	summaries := make([]models.StudentSummary, len(students))
	for i, s := range students {
		subjects, err := h.listSubjectsForStudent(s.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to load enrolled subjects")
		}
		summaries[i] = models.StudentSummary{Student: s, Subjects: subjects}
	}

	return c.JSON(summaries)
}

type createStudentRequest struct {
	Name  string  `json:"name"`
	Email *string `json:"email"`
}

func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
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

	student := models.Student{OwnerID: ownerID, Name: req.Name, Email: req.Email}
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

func (h *StudentHandler) Enroll(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

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
	if err := h.DB.Select("id").Where("id = ? AND owner_id = ?", studentID, ownerID).First(&student).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "student not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up student")
	}

	var subject models.Subject
	if err := h.DB.Select("id").Where("id = ? AND owner_id = ?", req.SubjectID, ownerID).First(&subject).Error; err != nil {
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

	subjects, err := h.listSubjectsForStudent(studentID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load enrolled subjects")
	}

	return c.Status(fiber.StatusCreated).JSON(subjects)
}

func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	studentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid student id")
	}

	var student models.Student
	err = h.DB.Where("id = ? AND owner_id = ?", studentID, ownerID).First(&student).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "student not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up student")
	}

	detail := models.StudentDetail{Student: student}

	detail.Subjects, err = h.listSubjectsForStudent(studentID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load enrolled subjects")
	}

	detail.Submissions, err = h.listWorkForStudent(studentID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load student work")
	}

	return c.JSON(detail)
}

func (h *StudentHandler) listSubjectsForStudent(studentID uuid.UUID) ([]models.Subject, error) {
	subjects := []models.Subject{}
	err := h.DB.Joins("JOIN enrollments ON enrollments.subject_id = subjects.id").
		Where("enrollments.student_id = ?", studentID).
		Order("subjects.name ASC").
		Find(&subjects).Error
	return subjects, err
}

// listWorkForStudent returns every assignment in a subject the student is
// enrolled in, alongside the student's own submission for it (if they've
// started one) — so assignments they haven't touched yet still show up as
// outstanding work.
func (h *StudentHandler) listWorkForStudent(studentID uuid.UUID) ([]models.StudentSubmission, error) {
	var subjectIDs []uuid.UUID
	if err := h.DB.Model(&models.Enrollment{}).
		Where("student_id = ?", studentID).
		Pluck("subject_id", &subjectIDs).Error; err != nil {
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
			// student hasn't started this assignment yet
		default:
			return nil, err
		}

		work = append(work, w)
	}

	return work, nil
}
