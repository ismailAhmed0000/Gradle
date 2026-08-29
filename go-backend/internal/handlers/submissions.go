package handlers

import (
	"bytes"
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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gradle-go-backend/internal/middleware"
	"gradle-go-backend/internal/models"
	"gradle-go-backend/internal/queue"
	"gradle-go-backend/internal/storage"
)

const compositedURLExpiry = 15 * time.Minute

type SubmissionHandler struct {
	DB      *gorm.DB
	Storage *storage.S3Storage
	Queue   *queue.JobQueue
}

func NewSubmissionHandler(db *gorm.DB, s *storage.S3Storage, q *queue.JobQueue) *SubmissionHandler {
	return &SubmissionHandler{DB: db, Storage: s, Queue: q}
}

type createSubmissionRequest struct {
	StudentName string `json:"student_name"`
}

func (h *SubmissionHandler) Create(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

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

	var assignment models.Assignment
	if err := h.DB.Select("id").Where("id = ? AND owner_id = ?", assignmentID, ownerID).First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "assignment not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up assignment")
	}

	// If the teacher already has a student on their roster with this name,
	// link the submission to that student record so it shows up on the
	// student's work page; mobile submitters aren't required to be enrolled.
	var studentID *uuid.UUID
	var student models.Student
	err = h.DB.Where("owner_id = ? AND lower(name) = lower(?)", ownerID, req.StudentName).First(&student).Error
	switch {
	case err == nil:
		studentID = &student.ID
	case errors.Is(err, gorm.ErrRecordNotFound):
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up student")
	}

	// A student has exactly one answer sheet per assignment: re-submitting the
	// same name resumes their existing submission instead of creating another.
	s := models.Submission{
		AssignmentID: assignmentID,
		StudentName:  req.StudentName,
		StudentID:    studentID,
		Status:       models.SubmissionStatusPending,
	}
	if err := h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "assignment_id"}, {Name: "student_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"student_name", "student_id"}),
	}).Create(&s).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create submission")
	}

	var pageCount int64
	if err := h.DB.Model(&models.SubmissionPage{}).Where("submission_id = ?", s.ID).Count(&pageCount).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load submission")
	}

	return c.Status(fiber.StatusCreated).JSON(createSubmissionResponse{Submission: s, PageCount: int(pageCount)})
}

type createSubmissionResponse struct {
	models.Submission
	PageCount int `json:"page_count"`
}

func (h *SubmissionHandler) ListForAssignment(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	assignmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid assignment id")
	}

	var assignment models.Assignment
	if err := h.DB.Select("id").Where("id = ? AND owner_id = ?", assignmentID, ownerID).First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "assignment not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up assignment")
	}

	var subs []models.Submission
	if err := h.DB.Where("assignment_id = ?", assignmentID).
		Order("created_at DESC").
		Find(&subs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load submissions")
	}

	summaries := make([]models.SubmissionSummary, len(subs))
	for i, s := range subs {
		pageCount, regionsDone, regionsTotal, err := submissionCounts(h.DB, s.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to load submissions")
		}
		summaries[i] = models.SubmissionSummary{
			Submission:         s,
			PageCount:          pageCount,
			AnswerRegionsDone:  regionsDone,
			AnswerRegionsTotal: regionsTotal,
		}
	}

	return c.JSON(summaries)
}

func submissionCounts(db *gorm.DB, submissionID uuid.UUID) (pageCount, regionsDone, regionsTotal int, err error) {
	var pages, done, total int64
	if err = db.Model(&models.SubmissionPage{}).Where("submission_id = ?", submissionID).Count(&pages).Error; err != nil {
		return
	}
	if err = db.Model(&models.AnswerRegion{}).Where("submission_id = ?", submissionID).Count(&total).Error; err != nil {
		return
	}
	if err = db.Model(&models.AnswerRegion{}).
		Where("submission_id = ? AND status = ?", submissionID, models.AnswerRegionStatusDone).
		Count(&done).Error; err != nil {
		return
	}
	return int(pages), int(done), int(total), nil
}

type uploadPageResponse struct {
	Page                models.SubmissionPage `json:"page"`
	AnswerRegionsQueued int                   `json:"answer_regions_queued"`
}

func (h *SubmissionHandler) UploadPage(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

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

	var submission models.Submission
	err = h.DB.Joins("JOIN assignments ON assignments.id = submissions.assignment_id").
		Where("submissions.id = ? AND assignments.owner_id = ?", submissionID, ownerID).
		First(&submission).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "submission not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up submission")
	}
	assignmentID := submission.AssignmentID

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
		submissionID, assignmentID, pageNumber, key, float64(imgConfig.Width), float64(imgConfig.Height),
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
	submissionID, assignmentID uuid.UUID, pageNumber int, rawImagePath string,
	imageWidth, imageHeight float64,
) ([]uuid.UUID, models.SubmissionPage, error) {
	var answerRegionIDs []uuid.UUID
	var page models.SubmissionPage

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		page = models.SubmissionPage{
			SubmissionID: submissionID,
			PageNumber:   pageNumber,
			RawImagePath: rawImagePath,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "submission_id"}, {Name: "page_number"}},
			DoUpdates: clause.AssignmentColumns([]string{"raw_image_path"}),
		}).Create(&page).Error; err != nil {
			return err
		}

		var questions []models.Question
		if err := tx.Where(
			"assignment_id = ? AND has_defined_region = true AND page_number = ?", assignmentID, pageNumber,
		).Find(&questions).Error; err != nil {
			return err
		}

		answerRegionIDs = make([]uuid.UUID, 0, len(questions))
		for _, q := range questions {
			// question region_x/y/width/height are normalized (0-1) fractions of
			// the page; convert to absolute pixel coordinates against this
			// specific scanned image before storing, since that's what the
			// extraction worker crops with.
			region := models.AnswerRegion{
				SubmissionID:     submissionID,
				QuestionID:       q.ID,
				SourcePageID:     page.ID,
				CropX:            derefOr(q.RegionX, 0) * imageWidth,
				CropY:            derefOr(q.RegionY, 0) * imageHeight,
				CropWidth:        derefOr(q.RegionWidth, 0) * imageWidth,
				CropHeight:       derefOr(q.RegionHeight, 0) * imageHeight,
				Status:           models.AnswerRegionStatusPending,
				ExtractedInkPath: nil,
				ErrorMessage:     nil,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "submission_id"}, {Name: "question_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"source_page_id", "crop_x", "crop_y", "crop_width", "crop_height",
					"status", "extracted_ink_path", "error_message",
				}),
			}).Create(&region).Error; err != nil {
				return err
			}

			if err := tx.Create(&models.ProcessingJob{
				JobType:     models.JobTypeExtractInk,
				ReferenceID: region.ID,
				Status:      "queued",
			}).Error; err != nil {
				return err
			}

			answerRegionIDs = append(answerRegionIDs, region.ID)
		}

		if err := tx.Model(&models.Submission{}).
			Where("id = ? AND status = ?", submissionID, models.SubmissionStatusPending).
			Update("status", models.SubmissionStatusProcessing).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, models.SubmissionPage{}, err
	}

	return answerRegionIDs, page, nil
}

func derefOr(v *float64, fallback float64) float64 {
	if v == nil {
		return fallback
	}
	return *v
}

func (h *SubmissionHandler) Get(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	submissionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid submission id")
	}

	var submission models.Submission
	err = h.DB.Joins("JOIN assignments ON assignments.id = submissions.assignment_id").
		Where("submissions.id = ? AND assignments.owner_id = ?", submissionID, ownerID).
		First(&submission).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "submission not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up submission")
	}

	detail := models.SubmissionDetail{Submission: submission}

	detail.Pages, err = h.listPages(submissionID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load pages")
	}

	detail.AnswerRegions, err = h.listAnswerRegions(submissionID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load answer regions")
	}

	detail.Composited, err = h.latestComposited(submissionID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load composited document")
	}

	return c.JSON(detail)
}

func (h *SubmissionHandler) GetComposited(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	submissionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid submission id")
	}

	var owned int64
	if err := h.DB.Table("submissions").
		Joins("JOIN assignments ON assignments.id = submissions.assignment_id").
		Where("submissions.id = ? AND assignments.owner_id = ?", submissionID, ownerID).
		Count(&owned).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up submission")
	}
	if owned == 0 {
		return fiber.NewError(fiber.StatusNotFound, "submission not found")
	}

	doc, err := h.latestComposited(submissionID)
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

func (h *SubmissionHandler) listPages(submissionID uuid.UUID) ([]models.SubmissionPage, error) {
	pages := []models.SubmissionPage{}
	err := h.DB.Where("submission_id = ?", submissionID).Order("page_number ASC").Find(&pages).Error
	return pages, err
}

func (h *SubmissionHandler) listAnswerRegions(submissionID uuid.UUID) ([]models.AnswerRegion, error) {
	regions := []models.AnswerRegion{}
	err := h.DB.Joins("Question").
		Where("answer_regions.submission_id = ?", submissionID).
		Order(`"Question"."question_number" ASC`).
		Find(&regions).Error
	if err != nil {
		return nil, err
	}
	for i := range regions {
		if regions[i].Question != nil {
			regions[i].QuestionNumber = regions[i].Question.QuestionNumber
		}
	}
	return regions, nil
}

func (h *SubmissionHandler) latestComposited(submissionID uuid.UUID) (*models.CompositedDocument, error) {
	var doc models.CompositedDocument
	err := h.DB.Where("submission_id = ?", submissionID).Order("version DESC").First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &doc, nil
}
