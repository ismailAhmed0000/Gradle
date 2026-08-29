package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"gradle-go-backend/internal/auth"
	"gradle-go-backend/internal/config"
	"gradle-go-backend/internal/middleware"
	"gradle-go-backend/internal/models"
)

const pgUniqueViolation = "23505"

type AuthHandler struct {
	DB     *gorm.DB
	Config *config.Config
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{DB: db, Config: cfg}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return fiber.NewError(fiber.StatusBadRequest, "a valid email is required")
	}
	if len(req.Password) < auth.MinPasswordLength {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to process password")
	}

	user := models.User{
		Email:        req.Email,
		PasswordHash: &passwordHash,
		Role:         models.RoleTeacher,
	}
	if err := h.DB.Create(&user).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return fiber.NewError(fiber.StatusConflict, "an account with this email already exists")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create account")
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.Config.JWTSecret, h.Config.JWTExpiryHours)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to issue token")
	}

	return c.Status(fiber.StatusCreated).JSON(authResponse{Token: token, User: user})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	const invalidCredentialsMsg = "invalid email or password"

	var user models.User
	err := h.DB.Where("email = ?", req.Email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusUnauthorized, invalidCredentialsMsg)
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up account")
	}

	if user.PasswordHash == nil || !auth.CheckPassword(*user.PasswordHash, req.Password) {
		return fiber.NewError(fiber.StatusUnauthorized, invalidCredentialsMsg)
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.Config.JWTSecret, h.Config.JWTExpiryHours)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to issue token")
	}

	return c.JSON(authResponse{Token: token, User: user})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "account not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up account")
	}

	return c.JSON(user)
}
