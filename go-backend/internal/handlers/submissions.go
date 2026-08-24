package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gradle-go-backend/internal/middleware"
	"gradle-go-backend/internal/models"
	"gradle-go-backend/internal/queue"
	"gradle-go-backend/internal/storage"
)

const compositedURLExpiry = 15 * time.Minute

type SubmissionHandler struct {
	DB      *pgxpool.Pool
	Storage *storage.S3Storage
	Queue   *queue.JobQueue
}

func NewSubmissionHandler(db *pgxpool.Pool, s *storage.S3Storage, q *queue.JobQueue) *SubmissionHandler {
	return &SubmissionHandler{DB: db, Storage: s, Queue: q}
}

type createSubmissionRequest struct {
	StudentName string `json:"student_name"`
}

func (h *SubmissionHandler) Create(c *fiber.Ctx) error {
	ownerID := c.Locals(middleware.LocalsUserID)

	assignmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid assignment id")
	}

	var req createSubmissionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.StudentName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "student_name is required")
	}

	var exists bool
	err = h.DB.QueryRow(
		c.Context(),
		`SELECT EXISTS(SELECT 1 FROM assignments WHERE id = $1 AND owner_id = $2)`,
		assignmentID, ownerID,
	).Scan(&exists)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up assignment")
	}
	if !exists {
		return fiber.NewError(fiber.StatusNotFound, "assignment not found")
	}

	// A student has exactly one answer sheet per assignment: re-submitting the
	// same name resumes their existing submission instead of creating another.
	var s models.Submission
	err = h.DB.QueryRow(
		c.Context(),
		`INSERT INTO submissions (assignment_id, student_name) VALUES ($1, $2)
		 ON CONFLICT (assignment_id, student_name) DO UPDATE SET student_name = EXCLUDED.student_name
		 RETURNING id, assignment_id, student_name, status, created_at`,
		assignmentID, req.StudentName,
	).Scan(&s.ID, &s.AssignmentID, &s.StudentName, &s.Status, &s.CreatedAt)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create submission")
	}

	var pageCount int
	if err := h.DB.QueryRow(
		c.Context(),
		`SELECT COUNT(*) FROM submission_pages WHERE submission_id = $1`,
		s.ID,
	).Scan(&pageCount); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load submission")
	}

	return c.Status(fiber.StatusCreated).JSON(createSubmissionResponse{Submission: s, PageCount: pageCount})
}

type createSubmissionResponse struct {
	models.Submission
	PageCount int `json:"page_count"`
}

func (h *SubmissionHandler) ListForAssignment(c *fiber.Ctx) error {
	ownerID := c.Locals(middleware.LocalsUserID)

	assignmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid assignment id")
	}

	var exists bool
	err = h.DB.QueryRow(
		c.Context(),
		`SELECT EXISTS(SELECT 1 FROM assignments WHERE id = $1 AND owner_id = $2)`,
		assignmentID, ownerID,
	).Scan(&exists)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up assignment")
	}
	if !exists {
		return fiber.NewError(fiber.StatusNotFound, "assignment not found")
	}

	rows, err := h.DB.Query(
		c.Context(),
		`SELECT s.id, s.assignment_id, s.student_name, s.status, s.created_at,
		        COUNT(DISTINCT sp.id) AS page_count,
		        COUNT(ar.id) FILTER (WHERE ar.status = 'done') AS regions_done,
		        COUNT(ar.id) AS regions_total
		 FROM submissions s
		 LEFT JOIN submission_pages sp ON sp.submission_id = s.id
		 LEFT JOIN answer_regions ar ON ar.submission_id = s.id
		 WHERE s.assignment_id = $1
		 GROUP BY s.id
		 ORDER BY s.created_at DESC`,
		assignmentID,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load submissions")
	}
	defer rows.Close()

	submissions := []models.SubmissionSummary{}
	for rows.Next() {
		var s models.SubmissionSummary
		if err := rows.Scan(
			&s.ID, &s.AssignmentID, &s.StudentName, &s.Status, &s.CreatedAt,
			&s.PageCount, &s.AnswerRegionsDone, &s.AnswerRegionsTotal,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to load submissions")
		}
		submissions = append(submissions, s)
	}
	if err := rows.Err(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load submissions")
	}

	return c.JSON(submissions)
}

type uploadPageResponse struct {
	Page                models.SubmissionPage `json:"page"`
	AnswerRegionsQueued int                   `json:"answer_regions_queued"`
}

func (h *SubmissionHandler) UploadPage(c *fiber.Ctx) error {
	ownerID := c.Locals(middleware.LocalsUserID)

	submissionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid submission id")
	}

	pageNumber, err := strconv.Atoi(c.FormValue("page_number"))
	if err != nil || pageNumber < 1 {
		return fiber.NewError(fiber.StatusBadRequest, "page_number must be a positive integer")
	}

	fileHeader, err := c.FormFile("page")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "page file is required")
	}

	var assignmentID uuid.UUID
	err = h.DB.QueryRow(
		c.Context(),
		`SELECT s.assignment_id FROM submissions s
		 JOIN assignments a ON a.id = s.assignment_id
		 WHERE s.id = $1 AND a.owner_id = $2`,
		submissionID, ownerID,
	).Scan(&assignmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "submission not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up submission")
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

	imgConfig, _, err := image.DecodeConfig(bytes.NewReader(fileBytes))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "uploaded file is not a valid image")
	}

	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	key := fmt.Sprintf("submissions/%s/pages/%d%s", submissionID, pageNumber, ext)

	if err := h.Storage.PutObject(c.Context(), key, bytes.NewReader(fileBytes), int64(len(fileBytes)), contentType); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to store scanned page")
	}

	answerRegionIDs, page, err := h.recordPageAndAnswerRegions(
		c.Context(), submissionID, assignmentID, pageNumber, key, float64(imgConfig.Width), float64(imgConfig.Height),
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save scanned page")
	}

	for _, regionID := range answerRegionIDs {
		if err := h.Queue.Push(c.Context(), models.JobTypeExtractInk, regionID.String()); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to queue extraction job")
		}
	}

	return c.Status(fiber.StatusCreated).JSON(uploadPageResponse{
		Page:                page,
		AnswerRegionsQueued: len(answerRegionIDs),
	})
}

func (h *SubmissionHandler) recordPageAndAnswerRegions(
	ctx context.Context, submissionID, assignmentID uuid.UUID, pageNumber int, rawImagePath string,
	imageWidth, imageHeight float64,
) ([]uuid.UUID, models.SubmissionPage, error) {
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		return nil, models.SubmissionPage{}, err
	}
	defer tx.Rollback(ctx)

	var page models.SubmissionPage
	err = tx.QueryRow(
		ctx,
		`INSERT INTO submission_pages (submission_id, page_number, raw_image_path)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (submission_id, page_number) DO UPDATE SET raw_image_path = EXCLUDED.raw_image_path
		 RETURNING id, submission_id, page_number, raw_image_path, created_at`,
		submissionID, pageNumber, rawImagePath,
	).Scan(&page.ID, &page.SubmissionID, &page.PageNumber, &page.RawImagePath, &page.CreatedAt)
	if err != nil {
		return nil, models.SubmissionPage{}, err
	}

	rows, err := tx.Query(
		ctx,
		`SELECT id, region_x, region_y, region_width, region_height FROM questions
		 WHERE assignment_id = $1 AND has_defined_region = true AND page_number = $2`,
		assignmentID, pageNumber,
	)
	if err != nil {
		return nil, models.SubmissionPage{}, err
	}

	type regionSeed struct {
		questionID                          uuid.UUID
		cropX, cropY, cropWidth, cropHeight float64
	}
	var seeds []regionSeed
	for rows.Next() {
		var s regionSeed
		if err := rows.Scan(&s.questionID, &s.cropX, &s.cropY, &s.cropWidth, &s.cropHeight); err != nil {
			rows.Close()
			return nil, models.SubmissionPage{}, err
		}
		seeds = append(seeds, s)
	}
	rowErr := rows.Err()
	rows.Close()
	if rowErr != nil {
		return nil, models.SubmissionPage{}, rowErr
	}

	answerRegionIDs := make([]uuid.UUID, 0, len(seeds))
	for _, s := range seeds {
		// question region_x/y/width/height are normalized (0-1) fractions of
		// the page; convert to absolute pixel coordinates against this
		// specific scanned image before storing, since that's what the
		// extraction worker crops with.
		cropX := s.cropX * imageWidth
		cropY := s.cropY * imageHeight
		cropWidth := s.cropWidth * imageWidth
		cropHeight := s.cropHeight * imageHeight

		var regionID uuid.UUID
		err = tx.QueryRow(
			ctx,
			`INSERT INTO answer_regions (submission_id, question_id, source_page_id, crop_x, crop_y, crop_width, crop_height, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')
			 ON CONFLICT ON CONSTRAINT answer_regions_submission_id_question_id_key
			 DO UPDATE SET source_page_id = EXCLUDED.source_page_id, crop_x = EXCLUDED.crop_x, crop_y = EXCLUDED.crop_y,
			               crop_width = EXCLUDED.crop_width, crop_height = EXCLUDED.crop_height,
			               status = 'pending', extracted_ink_path = NULL, error_message = NULL
			 RETURNING id`,
			submissionID, s.questionID, page.ID, cropX, cropY, cropWidth, cropHeight,
		).Scan(&regionID)
		if err != nil {
			return nil, models.SubmissionPage{}, err
		}

		if _, err := tx.Exec(
			ctx,
			`INSERT INTO processing_jobs (job_type, reference_id, status) VALUES ('extract_ink', $1, 'queued')`,
			regionID,
		); err != nil {
			return nil, models.SubmissionPage{}, err
		}

		answerRegionIDs = append(answerRegionIDs, regionID)
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE submissions SET status = 'processing' WHERE id = $1 AND status = 'pending'`,
		submissionID,
	); err != nil {
		return nil, models.SubmissionPage{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, models.SubmissionPage{}, err
	}

	return answerRegionIDs, page, nil
}

func (h *SubmissionHandler) Get(c *fiber.Ctx) error {
	ownerID := c.Locals(middleware.LocalsUserID)

	submissionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid submission id")
	}

	var detail models.SubmissionDetail
	err = h.DB.QueryRow(
		c.Context(),
		`SELECT s.id, s.assignment_id, s.student_name, s.status, s.created_at
		 FROM submissions s JOIN assignments a ON a.id = s.assignment_id
		 WHERE s.id = $1 AND a.owner_id = $2`,
		submissionID, ownerID,
	).Scan(&detail.ID, &detail.AssignmentID, &detail.StudentName, &detail.Status, &detail.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "submission not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up submission")
	}

	detail.Pages, err = h.listPages(c.Context(), submissionID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load pages")
	}

	detail.AnswerRegions, err = h.listAnswerRegions(c.Context(), submissionID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load answer regions")
	}

	detail.Composited, err = h.latestComposited(c.Context(), submissionID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load composited document")
	}

	return c.JSON(detail)
}

func (h *SubmissionHandler) GetComposited(c *fiber.Ctx) error {
	ownerID := c.Locals(middleware.LocalsUserID)

	submissionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid submission id")
	}

	var owned bool
	err = h.DB.QueryRow(
		c.Context(),
		`SELECT EXISTS(
		   SELECT 1 FROM submissions s JOIN assignments a ON a.id = s.assignment_id
		   WHERE s.id = $1 AND a.owner_id = $2
		 )`,
		submissionID, ownerID,
	).Scan(&owned)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up submission")
	}
	if !owned {
		return fiber.NewError(fiber.StatusNotFound, "submission not found")
	}

	doc, err := h.latestComposited(c.Context(), submissionID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load composited document")
	}
	if doc == nil {
		return fiber.NewError(fiber.StatusNotFound, "no composited document yet")
	}

	response := fiber.Map{"status": doc.Status}
	if doc.Status == models.CompositedDocumentStatusDone && doc.FilePath != nil {
		url, err := h.Storage.PresignedGetURL(c.Context(), *doc.FilePath, compositedURLExpiry)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to generate download link")
		}
		response["download_url"] = url
	}
	if doc.ErrorMessage != nil {
		response["error_message"] = *doc.ErrorMessage
	}

	return c.JSON(response)
}

func (h *SubmissionHandler) listPages(ctx context.Context, submissionID uuid.UUID) ([]models.SubmissionPage, error) {
	rows, err := h.DB.Query(
		ctx,
		`SELECT id, submission_id, page_number, raw_image_path, created_at
		 FROM submission_pages WHERE submission_id = $1 ORDER BY page_number ASC`,
		submissionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pages := []models.SubmissionPage{}
	for rows.Next() {
		var p models.SubmissionPage
		if err := rows.Scan(&p.ID, &p.SubmissionID, &p.PageNumber, &p.RawImagePath, &p.CreatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (h *SubmissionHandler) listAnswerRegions(ctx context.Context, submissionID uuid.UUID) ([]models.AnswerRegion, error) {
	rows, err := h.DB.Query(
		ctx,
		`SELECT ar.id, ar.submission_id, ar.question_id, q.question_number, ar.source_page_id, ar.status,
		        ar.crop_x, ar.crop_y, ar.crop_width, ar.crop_height, ar.extracted_ink_path, ar.error_message, ar.created_at
		 FROM answer_regions ar JOIN questions q ON q.id = ar.question_id
		 WHERE ar.submission_id = $1 ORDER BY q.question_number ASC`,
		submissionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	regions := []models.AnswerRegion{}
	for rows.Next() {
		var r models.AnswerRegion
		if err := rows.Scan(
			&r.ID, &r.SubmissionID, &r.QuestionID, &r.QuestionNumber, &r.SourcePageID, &r.Status,
			&r.CropX, &r.CropY, &r.CropWidth, &r.CropHeight, &r.ExtractedInkPath, &r.ErrorMessage, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		regions = append(regions, r)
	}
	return regions, rows.Err()
}

func (h *SubmissionHandler) latestComposited(ctx context.Context, submissionID uuid.UUID) (*models.CompositedDocument, error) {
	var doc models.CompositedDocument
	err := h.DB.QueryRow(
		ctx,
		`SELECT id, submission_id, version, status, file_path, error_message, created_at
		 FROM composited_documents WHERE submission_id = $1 ORDER BY version DESC LIMIT 1`,
		submissionID,
	).Scan(&doc.ID, &doc.SubmissionID, &doc.Version, &doc.Status, &doc.FilePath, &doc.ErrorMessage, &doc.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &doc, nil
}
