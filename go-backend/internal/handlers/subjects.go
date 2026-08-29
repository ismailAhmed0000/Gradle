package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"gradle-go-backend/internal/middleware"
	"gradle-go-backend/internal/models"
)

type SubjectHandler struct {
	DB *gorm.DB
}

func NewSubjectHandler(db *gorm.DB) *SubjectHandler {
	return &SubjectHandler{DB: db}
}

// List returns every subject for an admin (who needs the full picture to
// enroll students anywhere) or just the requester's own for a teacher.
func (h *SubjectHandler) List(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	query := h.DB.Model(&models.Subject{})
	if !middleware.IsAdmin(c) {
		query = query.Where("owner_id = ?", ownerID)
	}

	subjects := []models.Subject{}
	if err := query.Order("name ASC").Find(&subjects).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list subjects")
	}

	return c.JSON(subjects)
}

type createSubjectRequest struct {
	Name string `json:"name"`
}

func (h *SubjectHandler) Create(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	var req createSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	subject := models.Subject{OwnerID: ownerID, Name: req.Name}
	if err := h.DB.Create(&subject).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return fiber.NewError(fiber.StatusConflict, "a subject with this name already exists")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create subject")
	}

	return c.Status(fiber.StatusCreated).JSON(subject)
}

// GetByID returns a subject and its enrolled students. A teacher gets 404
// (not 403) for a subject they don't own, same as if it didn't exist.
func (h *SubjectHandler) GetByID(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	subjectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid subject id")
	}

	var subject models.Subject
	if err := h.DB.First(&subject, "id = ?", subjectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "subject not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up subject")
	}
	if !middleware.IsAdmin(c) && subject.OwnerID != userID {
		return fiber.NewError(fiber.StatusNotFound, "subject not found")
	}

	students := []models.Student{}
	if err := h.DB.Joins("JOIN enrollments ON enrollments.student_id = students.id").
		Where("enrollments.subject_id = ?", subjectID).
		Order("students.name ASC").
		Find(&students).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load students")
	}

	return c.JSON(models.SubjectDetail{Subject: subject, Students: students})
}
