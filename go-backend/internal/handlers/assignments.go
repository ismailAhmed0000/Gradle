package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gradle-go-backend/internal/middleware"
	"gradle-go-backend/internal/models"
	"gradle-go-backend/internal/storage"
)

const assignmentFileURLExpiry = 15 * time.Minute

type AssignmentHandler struct {
	DB      *pgxpool.Pool
	Storage *storage.S3Storage
}

func NewAssignmentHandler(db *pgxpool.Pool, s *storage.S3Storage) *AssignmentHandler {
	return &AssignmentHandler{DB: db, Storage: s}
}

func (h *AssignmentHandler) List(c *fiber.Ctx) error {
	ownerID := c.Locals(middleware.LocalsUserID)

	rows, err := h.DB.Query(
		c.Context(),
		`SELECT id, owner_id, title, created_at FROM assignments
		 WHERE owner_id = $1 ORDER BY created_at DESC`,
		ownerID,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list assignments")
	}
	defer rows.Close()

	assignments := []models.Assignment{}
	for rows.Next() {
		var a models.Assignment
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Title, &a.CreatedAt); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to read assignments")
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to read assignments")
	}

	return c.JSON(assignments)
}

func (h *AssignmentHandler) GetByID(c *fiber.Ctx) error {
	ownerID := c.Locals(middleware.LocalsUserID)

	assignmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid assignment id")
	}

	var detail models.AssignmentDetail
	err = h.DB.QueryRow(
		c.Context(),
		`SELECT id, owner_id, title, created_at FROM assignments WHERE id = $1 AND owner_id = $2`,
		assignmentID, ownerID,
	).Scan(&detail.ID, &detail.OwnerID, &detail.Title, &detail.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "assignment not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up assignment")
	}

	detail.Questions, err = h.listQuestions(c.Context(), assignmentID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load questions")
	}

	detail.AssignmentFiles, err = h.listAssignmentFiles(c.Context(), assignmentID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load assignment files")
	}

	return c.JSON(detail)
}

func (h *AssignmentHandler) listQuestions(ctx context.Context, assignmentID uuid.UUID) ([]models.Question, error) {
	rows, err := h.DB.Query(
		ctx,
		`SELECT id, assignment_id, assignment_file_id, question_number, has_defined_region,
		        page_number, region_x, region_y, region_width, region_height, created_at
		 FROM questions WHERE assignment_id = $1 ORDER BY question_number ASC`,
		assignmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	questions := []models.Question{}
	for rows.Next() {
		var q models.Question
		if err := rows.Scan(
			&q.ID, &q.AssignmentID, &q.AssignmentFileID, &q.QuestionNumber, &q.HasDefinedRegion,
			&q.PageNumber, &q.RegionX, &q.RegionY, &q.RegionWidth, &q.RegionHeight, &q.CreatedAt,
		); err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	return questions, rows.Err()
}

func (h *AssignmentHandler) listAssignmentFiles(ctx context.Context, assignmentID uuid.UUID) ([]models.AssignmentFile, error) {
	rows, err := h.DB.Query(
		ctx,
		`SELECT id, assignment_id, file_path, page_count, created_at
		 FROM assignment_files WHERE assignment_id = $1 ORDER BY created_at ASC`,
		assignmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []models.AssignmentFile{}
	for rows.Next() {
		var f models.AssignmentFile
		if err := rows.Scan(&f.ID, &f.AssignmentID, &f.FilePath, &f.PageCount, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range files {
		url, err := h.Storage.PresignedGetURL(ctx, files[i].FilePath, assignmentFileURLExpiry)
		if err != nil {
			return nil, err
		}
		files[i].DownloadURL = url
	}

	return files, nil
}
