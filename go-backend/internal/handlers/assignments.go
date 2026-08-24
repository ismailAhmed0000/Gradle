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
		`SELECT a.id, a.owner_id, a.title, a.subject, a.due_date, a.created_at, latest.status
		 FROM assignments a
		 LEFT JOIN LATERAL (
		     SELECT s.status FROM submissions s
		     WHERE s.assignment_id = a.id
		     ORDER BY s.created_at DESC LIMIT 1
		 ) latest ON true
		 WHERE a.owner_id = $1 ORDER BY a.created_at DESC`,
		ownerID,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list assignments")
	}
	defer rows.Close()

	assignments := []models.Assignment{}
	for rows.Next() {
		var a models.Assignment
		var latestSubmissionStatus *string
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Title, &a.Subject, &a.DueDate, &a.CreatedAt, &latestSubmissionStatus); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to read assignments")
		}
		a.Status = computeAssignmentStatus(a.DueDate, latestSubmissionStatus)
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to read assignments")
	}

	return c.JSON(assignments)
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

func (h *AssignmentHandler) GetByID(c *fiber.Ctx) error {
	ownerID := c.Locals(middleware.LocalsUserID)

	assignmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid assignment id")
	}

	var detail models.AssignmentDetail
	err = h.DB.QueryRow(
		c.Context(),
		`SELECT a.id, a.owner_id, a.title, a.subject, a.due_date, a.created_at, u.email
		 FROM assignments a JOIN users u ON u.id = a.owner_id
		 WHERE a.id = $1 AND a.owner_id = $2`,
		assignmentID, ownerID,
	).Scan(
		&detail.ID, &detail.OwnerID, &detail.Title, &detail.Subject, &detail.DueDate, &detail.CreatedAt,
		&detail.TeacherEmail,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "assignment not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up assignment")
	}

	var latestSubmissionStatus *string
	err = h.DB.QueryRow(
		c.Context(),
		`SELECT status FROM submissions WHERE assignment_id = $1 ORDER BY created_at DESC LIMIT 1`,
		assignmentID,
	).Scan(&latestSubmissionStatus)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up submission status")
	}
	detail.Status = computeAssignmentStatus(detail.DueDate, latestSubmissionStatus)

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
