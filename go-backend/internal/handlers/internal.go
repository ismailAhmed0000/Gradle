package handlers

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gradle-go-backend/internal/models"
	"gradle-go-backend/internal/queue"
)

type InternalHandler struct {
	DB    *pgxpool.Pool
	Queue *queue.JobQueue
}

func NewInternalHandler(db *pgxpool.Pool, q *queue.JobQueue) *InternalHandler {
	return &InternalHandler{DB: db, Queue: q}
}

// --- answer regions (extract_ink jobs) ---

func (h *InternalHandler) AnswerRegionContext(c *fiber.Ctx) error {
	regionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid answer region id")
	}

	var cropX, cropY, cropWidth, cropHeight float64
	var rawImagePath string
	err = h.DB.QueryRow(
		c.Context(),
		`SELECT ar.crop_x, ar.crop_y, ar.crop_width, ar.crop_height, sp.raw_image_path
		 FROM answer_regions ar JOIN submission_pages sp ON sp.id = ar.source_page_id
		 WHERE ar.id = $1`,
		regionID,
	).Scan(&cropX, &cropY, &cropWidth, &cropHeight, &rawImagePath)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "answer region not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up answer region")
	}

	return c.JSON(fiber.Map{
		"source_page": fiber.Map{"raw_image_path": rawImagePath},
		"crop_x":      cropX,
		"crop_y":      cropY,
		"crop_width":  cropWidth,
		"crop_height": cropHeight,
	})
}

func (h *InternalHandler) StartAnswerRegion(c *fiber.Ctx) error {
	regionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid answer region id")
	}

	if _, err := h.DB.Exec(
		c.Context(),
		`UPDATE answer_regions SET status = 'processing' WHERE id = $1`,
		regionID,
	); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to start answer region job")
	}
	if _, err := h.DB.Exec(
		c.Context(),
		`UPDATE processing_jobs SET status = 'running', updated_at = now() WHERE job_type = 'extract_ink' AND reference_id = $1`,
		regionID,
	); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update job status")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

type reportAnswerRegionRequest struct {
	Status           string  `json:"status"`
	ExtractedInkPath *string `json:"extracted_ink_path"`
	ErrorMessage     *string `json:"error_message"`
}

func (h *InternalHandler) ReportAnswerRegionResult(c *fiber.Ctx) error {
	regionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid answer region id")
	}

	var req reportAnswerRegionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var submissionID uuid.UUID
	err = h.DB.QueryRow(c.Context(), `SELECT submission_id FROM answer_regions WHERE id = $1`, regionID).Scan(&submissionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "answer region not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up answer region")
	}

	switch req.Status {
	case "done":
		if _, err := h.DB.Exec(
			c.Context(),
			`UPDATE answer_regions SET status = 'done', extracted_ink_path = $1, error_message = NULL WHERE id = $2`,
			req.ExtractedInkPath, regionID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update answer region")
		}
		if _, err := h.DB.Exec(
			c.Context(),
			`UPDATE processing_jobs SET status = 'done', updated_at = now() WHERE job_type = 'extract_ink' AND reference_id = $1`,
			regionID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update job status")
		}
		compositedID, err := h.maybeStartCompositing(c.Context(), submissionID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to start compositing")
		}
		if compositedID != nil {
			if err := h.Queue.Push(c.Context(), models.JobTypeCompositePDF, compositedID.String()); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "failed to queue compositing job")
			}
		}
	case "failed":
		if _, err := h.DB.Exec(
			c.Context(),
			`UPDATE answer_regions SET status = 'failed', error_message = $1 WHERE id = $2`,
			req.ErrorMessage, regionID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update answer region")
		}
		if _, err := h.DB.Exec(
			c.Context(),
			`UPDATE processing_jobs SET status = 'failed', error_message = $1, updated_at = now()
			 WHERE job_type = 'extract_ink' AND reference_id = $2`,
			req.ErrorMessage, regionID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update job status")
		}
		if _, err := h.DB.Exec(
			c.Context(),
			`UPDATE submissions SET status = 'failed' WHERE id = $1`,
			submissionID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update submission")
		}
	default:
		return fiber.NewError(fiber.StatusBadRequest, "status must be 'done' or 'failed'")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// maybeStartCompositing creates a composited_documents row once every answer
// region for the submission has finished extraction, returning its id so the
// caller can enqueue the composite_pdf job outside the DB transaction.
func (h *InternalHandler) maybeStartCompositing(ctx context.Context, submissionID uuid.UUID) (*uuid.UUID, error) {
	var remaining, total int
	err := h.DB.QueryRow(
		ctx,
		`SELECT COUNT(*) FILTER (WHERE status != 'done'), COUNT(*) FROM answer_regions WHERE submission_id = $1`,
		submissionID,
	).Scan(&remaining, &total)
	if err != nil {
		return nil, err
	}
	if total == 0 || remaining > 0 {
		return nil, nil
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var nextVersion int
	if err := tx.QueryRow(
		ctx,
		`SELECT COALESCE(MAX(version), 0) + 1 FROM composited_documents WHERE submission_id = $1`,
		submissionID,
	).Scan(&nextVersion); err != nil {
		return nil, err
	}

	var compositedID uuid.UUID
	if err := tx.QueryRow(
		ctx,
		`INSERT INTO composited_documents (submission_id, version, status) VALUES ($1, $2, 'pending') RETURNING id`,
		submissionID, nextVersion,
	).Scan(&compositedID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO processing_jobs (job_type, reference_id, status) VALUES ('composite_pdf', $1, 'queued')`,
		compositedID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &compositedID, nil
}

// --- composited documents (composite_pdf jobs) ---

type compositeAnswer struct {
	AssignmentFileID uuid.UUID `json:"assignment_file_id"`
	ExtractedInkPath string    `json:"extracted_ink_path"`
	HasDefinedRegion bool      `json:"has_defined_region"`
	PageNumber       *int      `json:"page_number"`
	RegionX          *float64  `json:"region_x"`
	RegionY          *float64  `json:"region_y"`
	RegionWidth      *float64  `json:"region_width"`
	RegionHeight     *float64  `json:"region_height"`
	QuestionNumber   int       `json:"question_number"`
	QuestionID       uuid.UUID `json:"question_id"`
}

type compositeAssignmentFile struct {
	ID       uuid.UUID `json:"id"`
	FilePath string    `json:"file_path"`
}

func (h *InternalHandler) CompositeContext(c *fiber.Ctx) error {
	compositedID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid composited document id")
	}

	var submissionID uuid.UUID
	var version int
	var assignmentID uuid.UUID
	err = h.DB.QueryRow(
		c.Context(),
		`SELECT cd.submission_id, cd.version, s.assignment_id
		 FROM composited_documents cd JOIN submissions s ON s.id = cd.submission_id
		 WHERE cd.id = $1`,
		compositedID,
	).Scan(&submissionID, &version, &assignmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "composited document not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up composited document")
	}

	answerRows, err := h.DB.Query(
		c.Context(),
		`SELECT q.assignment_file_id, ar.extracted_ink_path, q.has_defined_region,
		        q.page_number, q.region_x, q.region_y, q.region_width, q.region_height,
		        q.question_number, q.id
		 FROM answer_regions ar JOIN questions q ON q.id = ar.question_id
		 WHERE ar.submission_id = $1 AND ar.status = 'done'
		 ORDER BY q.question_number ASC`,
		submissionID,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load answers")
	}
	answers := []compositeAnswer{}
	for answerRows.Next() {
		var a compositeAnswer
		var inkPath *string
		if err := answerRows.Scan(
			&a.AssignmentFileID, &inkPath, &a.HasDefinedRegion,
			&a.PageNumber, &a.RegionX, &a.RegionY, &a.RegionWidth, &a.RegionHeight,
			&a.QuestionNumber, &a.QuestionID,
		); err != nil {
			answerRows.Close()
			return fiber.NewError(fiber.StatusInternalServerError, "failed to read answers")
		}
		if inkPath != nil {
			a.ExtractedInkPath = *inkPath
		}
		answers = append(answers, a)
	}
	rowErr := answerRows.Err()
	answerRows.Close()
	if rowErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to read answers")
	}

	fileRows, err := h.DB.Query(
		c.Context(),
		`SELECT id, file_path FROM assignment_files WHERE assignment_id = $1`,
		assignmentID,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load assignment files")
	}
	files := []compositeAssignmentFile{}
	for fileRows.Next() {
		var f compositeAssignmentFile
		if err := fileRows.Scan(&f.ID, &f.FilePath); err != nil {
			fileRows.Close()
			return fiber.NewError(fiber.StatusInternalServerError, "failed to read assignment files")
		}
		files = append(files, f)
	}
	rowErr = fileRows.Err()
	fileRows.Close()
	if rowErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to read assignment files")
	}

	return c.JSON(fiber.Map{
		"submission_id":    submissionID,
		"version":          version,
		"answers":          answers,
		"assignment_files": files,
	})
}

func (h *InternalHandler) StartComposite(c *fiber.Ctx) error {
	compositedID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid composited document id")
	}

	if _, err := h.DB.Exec(
		c.Context(),
		`UPDATE composited_documents SET status = 'generating' WHERE id = $1`,
		compositedID,
	); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to start compositing job")
	}
	if _, err := h.DB.Exec(
		c.Context(),
		`UPDATE processing_jobs SET status = 'running', updated_at = now() WHERE job_type = 'composite_pdf' AND reference_id = $1`,
		compositedID,
	); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update job status")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

type compositePageReport struct {
	PageNumber int        `json:"page_number"`
	PageType   string     `json:"page_type"`
	QuestionID *uuid.UUID `json:"question_id"`
}

type reportCompositeRequest struct {
	Status       string                `json:"status"`
	FilePath     *string               `json:"file_path"`
	Pages        []compositePageReport `json:"pages"`
	ErrorMessage *string               `json:"error_message"`
}

func (h *InternalHandler) ReportCompositeResult(c *fiber.Ctx) error {
	compositedID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid composited document id")
	}

	var req reportCompositeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var submissionID uuid.UUID
	err = h.DB.QueryRow(
		c.Context(), `SELECT submission_id FROM composited_documents WHERE id = $1`, compositedID,
	).Scan(&submissionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "composited document not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up composited document")
	}

	switch req.Status {
	case "done":
		tx, err := h.DB.Begin(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update composited document")
		}
		defer tx.Rollback(c.Context())

		if _, err := tx.Exec(
			c.Context(),
			`UPDATE composited_documents SET status = 'done', file_path = $1, error_message = NULL WHERE id = $2`,
			req.FilePath, compositedID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update composited document")
		}
		for _, p := range req.Pages {
			if _, err := tx.Exec(
				c.Context(),
				`INSERT INTO composited_document_pages (composited_document_id, page_number, page_type, question_id)
				 VALUES ($1, $2, $3, $4)`,
				compositedID, p.PageNumber, p.PageType, p.QuestionID,
			); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "failed to record composited pages")
			}
		}
		if _, err := tx.Exec(
			c.Context(), `UPDATE submissions SET status = 'composited' WHERE id = $1`, submissionID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update submission")
		}
		if err := tx.Commit(c.Context()); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update composited document")
		}
		if _, err := h.DB.Exec(
			c.Context(),
			`UPDATE processing_jobs SET status = 'done', updated_at = now() WHERE job_type = 'composite_pdf' AND reference_id = $1`,
			compositedID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update job status")
		}
	case "failed":
		if _, err := h.DB.Exec(
			c.Context(),
			`UPDATE composited_documents SET status = 'failed', error_message = $1 WHERE id = $2`,
			req.ErrorMessage, compositedID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update composited document")
		}
		if _, err := h.DB.Exec(
			c.Context(),
			`UPDATE processing_jobs SET status = 'failed', error_message = $1, updated_at = now()
			 WHERE job_type = 'composite_pdf' AND reference_id = $2`,
			req.ErrorMessage, compositedID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update job status")
		}
		if _, err := h.DB.Exec(
			c.Context(), `UPDATE submissions SET status = 'failed' WHERE id = $1`, submissionID,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update submission")
		}
	default:
		return fiber.NewError(fiber.StatusBadRequest, "status must be 'done' or 'failed'")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
