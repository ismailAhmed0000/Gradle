package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"gorm.io/gorm"

	"gradle-go-backend/internal/middleware"
	"gradle-go-backend/internal/models"
	"gradle-go-backend/internal/storage"
)

const assignmentFileURLExpiry = 15 * time.Minute

type AssignmentHandler struct {
	DB      *gorm.DB
	Storage *storage.S3Storage
}

func NewAssignmentHandler(db *gorm.DB, s *storage.S3Storage) *AssignmentHandler {
	return &AssignmentHandler{DB: db, Storage: s}
}

func (h *AssignmentHandler) List(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	query := h.DB.Preload("Subject")
	if middleware.IsStudent(c) {
		student, err := studentForUser(h.DB, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to look up student record")
		}
		if student == nil {
			return c.JSON([]models.Assignment{})
		}
		query = query.
			Joins("JOIN enrollments ON enrollments.subject_id = assignments.subject_id").
			Where("enrollments.student_id = ?", student.ID)
	} else {
		query = query.Where("owner_id = ?", userID)
	}

	var assignments []models.Assignment
	if err := query.
		Order("created_at DESC").
		Find(&assignments).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list assignments")
	}

	// A student's own status is scoped to their own submission — the
	// teacher's list view intentionally keeps showing whichever submission
	// was created most recently across the whole class (pre-existing
	// behavior for the "who needs grading" glance).
	var studentScope *uuid.UUID
	if middleware.IsStudent(c) {
		student, err := studentForUser(h.DB, userID)
		if err == nil && student != nil {
			studentScope = &student.ID
		}
	}

	for i := range assignments {
		if err := h.attachComputedFields(&assignments[i], studentScope); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to read assignments")
		}
	}

	return c.JSON(assignments)
}

func (h *AssignmentHandler) attachComputedFields(a *models.Assignment, studentID *uuid.UUID) error {
	if a.Subject != nil {
		a.SubjectName = &a.Subject.Name
	}

	query := h.DB.Where("assignment_id = ?", a.ID)
	if studentID != nil {
		query = query.Where("student_id = ?", *studentID)
	}
	var latest models.Submission
	err := query.Order("created_at DESC").First(&latest).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var latestStatus *string
	if err == nil {
		s := string(latest.Status)
		latestStatus = &s
	}
	a.Status = computeAssignmentStatus(a.DueDate, latestStatus)
	return nil
}

// computeAssignmentStatus derives a single status for an assignment from the
// most recently created submission (a student has exactly one answer paper
// per assignment) and the assignment's due date.
func computeAssignmentStatus(dueDate *time.Time, latestSubmissionStatus *string) models.AssignmentStatus {
	if latestSubmissionStatus != nil {
		if models.SubmissionStatus(*latestSubmissionStatus) == models.SubmissionStatusComposited {
			return models.AssignmentStatusGraded
		}
		return models.AssignmentStatusSubmitted
	}
	if dueDate != nil && dueDate.Before(time.Now()) {
		return models.AssignmentStatusExpired
	}
	return models.AssignmentStatusPending
}

type createAssignmentRequest struct {
	Title     string     `json:"title"`
	SubjectID uuid.UUID  `json:"subject_id"`
	DueDate   *time.Time `json:"due_date"`
}

func (h *AssignmentHandler) Create(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	var req createAssignmentRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Title == "" {
		return fiber.NewError(fiber.StatusBadRequest, "title is required")
	}
	if req.SubjectID == uuid.Nil {
		return fiber.NewError(fiber.StatusBadRequest, "subject_id is required")
	}

	var subject models.Subject
	if err := h.DB.Where("id = ? AND owner_id = ?", req.SubjectID, ownerID).First(&subject).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "subject not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up subject")
	}

	a := models.Assignment{
		OwnerID:   ownerID,
		Title:     req.Title,
		SubjectID: &req.SubjectID,
		DueDate:   req.DueDate,
		Source:    models.AssignmentSourceManual,
	}
	if err := h.DB.Create(&a).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create assignment")
	}

	a.SubjectName = &subject.Name
	a.Status = computeAssignmentStatus(a.DueDate, nil)

	return c.Status(fiber.StatusCreated).JSON(a)
}

func (h *AssignmentHandler) GetByID(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	assignmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid assignment id")
	}

	query := h.DB.Preload("Subject").Preload("Owner").Where("id = ?", assignmentID)

	var studentScope *uuid.UUID
	if middleware.IsStudent(c) {
		student, serr := studentForUser(h.DB, userID)
		if serr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to look up student record")
		}
		if student == nil {
			return fiber.NewError(fiber.StatusNotFound, "assignment not found")
		}
		studentScope = &student.ID
		query = query.
			Joins("JOIN enrollments ON enrollments.subject_id = assignments.subject_id").
			Where("enrollments.student_id = ?", student.ID)
	} else {
		query = query.Where("owner_id = ?", userID)
	}

	var assignment models.Assignment
	err = query.First(&assignment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "assignment not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up assignment")
	}
	if err := h.attachComputedFields(&assignment, studentScope); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up submission status")
	}

	detail := models.AssignmentDetail{Assignment: assignment}
	if assignment.Owner != nil {
		detail.TeacherEmail = assignment.Owner.Email
	}

	detail.Questions, err = h.listQuestions(assignmentID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load questions")
	}

	detail.AssignmentFiles, err = h.listAssignmentFiles(c, assignmentID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load assignment files")
	}

	return c.JSON(detail)
}

func (h *AssignmentHandler) listQuestions(assignmentID uuid.UUID) ([]models.Question, error) {
	questions := []models.Question{}
	err := h.DB.Where("assignment_id = ?", assignmentID).
		Order("question_number ASC").
		Find(&questions).Error
	return questions, err
}

func (h *AssignmentHandler) UploadFile(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	assignmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid assignment id")
	}

	var assignment models.Assignment
	if err := h.DB.Select("id", "source").Where("id = ? AND owner_id = ?", assignmentID, ownerID).First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "assignment not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up assignment")
	}
	if assignment.Source == models.AssignmentSourceClassroom {
		return fiber.NewError(fiber.StatusForbidden, "assignments imported from google classroom are read-only")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is required")
	}
	if !strings.EqualFold(filepath.Ext(fileHeader.Filename), ".pdf") {
		return fiber.NewError(fiber.StatusBadRequest, "file must be a PDF")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "failed to read uploaded file")
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "failed to read uploaded file")
	}

	pageCount, err := api.PageCount(bytes.NewReader(fileBytes), model.NewDefaultConfiguration())
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is not a valid PDF")
	}

	assignmentFile := models.AssignmentFile{
		ID:           uuid.New(),
		AssignmentID: assignmentID,
		PageCount:    pageCount,
	}
	assignmentFile.FilePath = fmt.Sprintf("assignments/%s/%s.pdf", assignmentID, assignmentFile.ID)

	if err := h.Storage.PutObject(
		c.Context(), assignmentFile.FilePath, bytes.NewReader(fileBytes), int64(len(fileBytes)), "application/pdf",
	); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to store file")
	}

	if err := h.DB.Create(&assignmentFile).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to record file")
	}

	url, err := h.Storage.PresignedGetURL(c.Context(), assignmentFile.FilePath, assignmentFileURLExpiry)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate download link")
	}
	assignmentFile.DownloadURL = url

	return c.Status(fiber.StatusCreated).JSON(assignmentFile)
}

func (h *AssignmentHandler) listAssignmentFiles(c *fiber.Ctx, assignmentID uuid.UUID) ([]models.AssignmentFile, error) {
	files := []models.AssignmentFile{}
	if err := h.DB.Where("assignment_id = ?", assignmentID).
		Order("created_at ASC").
		Find(&files).Error; err != nil {
		return nil, err
	}

	for i := range files {
		url, err := h.Storage.PresignedGetURL(c.Context(), files[i].FilePath, assignmentFileURLExpiry)
		if err != nil {
			return nil, err
		}
		files[i].DownloadURL = url
	}

	return files, nil
}
