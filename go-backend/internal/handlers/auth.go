package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gradle-go-backend/internal/auth"
	"gradle-go-backend/internal/config"
	"gradle-go-backend/internal/models"
)

const pgUniqueViolation = "23505"

type AuthHandler struct {
	DB     *pgxpool.Pool
	Config *config.Config
}

func NewAuthHandler(db *pgxpool.Pool, cfg *config.Config) *AuthHandler {
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

	var user models.User
	err = h.DB.QueryRow(
		c.Context(),
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3)
		 RETURNING id, email, role, created_at`,
		req.Email, passwordHash, models.RoleTeacher,
	).Scan(&user.ID, &user.Email, &user.Role, &user.CreatedAt)

	if err != nil {
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
	var passwordHash string
	err := h.DB.QueryRow(
		c.Context(),
		`SELECT id, email, password_hash, role, created_at FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Email, &passwordHash, &user.Role, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusUnauthorized, invalidCredentialsMsg)
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up account")
	}

	if !auth.CheckPassword(passwordHash, req.Password) {
		return fiber.NewError(fiber.StatusUnauthorized, invalidCredentialsMsg)
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.Config.JWTSecret, h.Config.JWTExpiryHours)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to issue token")
	}

	return c.JSON(authResponse{Token: token, User: user})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := c.Locals("user_id")

	var user models.User
	err := h.DB.QueryRow(
		context.Background(),
		`SELECT id, email, role, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Role, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "account not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up account")
	}

	return c.JSON(user)
}
