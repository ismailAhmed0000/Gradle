package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
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

func (h *SubjectHandler) List(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	subjects := []models.Subject{}
	if err := h.DB.Where("owner_id = ?", ownerID).Order("name ASC").Find(&subjects).Error; err != nil {
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
