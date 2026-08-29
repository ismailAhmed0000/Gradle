package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"gradle-go-backend/internal/models"
	"gradle-go-backend/internal/queue"
)

type InternalHandler struct {
	DB    *gorm.DB
	Queue *queue.JobQueue
}

func NewInternalHandler(db *gorm.DB, q *queue.JobQueue) *InternalHandler {
	return &InternalHandler{DB: db, Queue: q}
}

// --- answer regions (extract_ink jobs) ---

func (h *InternalHandler) AnswerRegionContext(c *fiber.Ctx) error {
	regionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid answer region id")
	}

	var region models.AnswerRegion
	if err := h.DB.First(&region, "id = ?", regionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "answer region not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up answer region")
	}

	var page models.SubmissionPage
	if err := h.DB.First(&page, "id = ?", region.SourcePageID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up source page")
	}

	return c.JSON(fiber.Map{
		"source_page": fiber.Map{"raw_image_path": page.RawImagePath},
		"crop_x":      region.CropX,
		"crop_y":      region.CropY,
		"crop_width":  region.CropWidth,
		"crop_height": region.CropHeight,
	})
}

func (h *InternalHandler) StartAnswerRegion(c *fiber.Ctx) error {
	regionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid answer region id")
	}

	if err := h.DB.Model(&models.AnswerRegion{}).
		Where("id = ?", regionID).
		Update("status", models.AnswerRegionStatusProcessing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to start answer region job")
	}
	if err := h.DB.Model(&models.ProcessingJob{}).
		Where("job_type = ? AND reference_id = ?", models.JobTypeExtractInk, regionID).
		Update("status", "running").Error; err != nil {
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

	var region models.AnswerRegion
	if err := h.DB.Select("id", "submission_id").First(&region, "id = ?", regionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "answer region not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up answer region")
	}
	submissionID := region.SubmissionID

	switch req.Status {
	case "done":
		if err := h.DB.Model(&models.AnswerRegion{}).Where("id = ?", regionID).Updates(map[string]any{
			"status":             models.AnswerRegionStatusDone,
			"extracted_ink_path": req.ExtractedInkPath,
			"error_message":      nil,
		}).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update answer region")
		}
		if err := h.DB.Model(&models.ProcessingJob{}).
			Where("job_type = ? AND reference_id = ?", models.JobTypeExtractInk, regionID).
			Update("status", "done").Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update job status")
		}
		compositedID, err := h.maybeStartCompositing(submissionID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to start compositing")
		}
		if compositedID != nil {
			if err := h.Queue.Push(c.Context(), models.JobTypeCompositePDF, compositedID.String()); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "failed to queue compositing job")
			}
		}
	case "failed":
		if err := h.DB.Model(&models.AnswerRegion{}).Where("id = ?", regionID).Updates(map[string]any{
			"status":        models.AnswerRegionStatusFailed,
			"error_message": req.ErrorMessage,
		}).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update answer region")
		}
		if err := h.DB.Model(&models.ProcessingJob{}).
			Where("job_type = ? AND reference_id = ?", models.JobTypeExtractInk, regionID).
			Updates(map[string]any{"status": "failed", "error_message": req.ErrorMessage}).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update job status")
		}
		if err := h.DB.Model(&models.Submission{}).
			Where("id = ?", submissionID).
			Update("status", models.SubmissionStatusFailed).Error; err != nil {
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
func (h *InternalHandler) maybeStartCompositing(submissionID uuid.UUID) (*uuid.UUID, error) {
	var total, done int64
	if err := h.DB.Model(&models.AnswerRegion{}).Where("submission_id = ?", submissionID).Count(&total).Error; err != nil {
		return nil, err
	}
	if err := h.DB.Model(&models.AnswerRegion{}).
		Where("submission_id = ? AND status = ?", submissionID, models.AnswerRegionStatusDone).
		Count(&done).Error; err != nil {
		return nil, err
	}
	if total == 0 || done < total {
		return nil, nil
	}

	var compositedID uuid.UUID
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var latest models.CompositedDocument
		nextVersion := 1
		err := tx.Where("submission_id = ?", submissionID).Order("version DESC").First(&latest).Error
		switch {
		case err == nil:
			nextVersion = latest.Version + 1
		case errors.Is(err, gorm.ErrRecordNotFound):
			// first composited document for this submission
		default:
			return err
		}

		doc := models.CompositedDocument{
			SubmissionID: submissionID,
			Version:      nextVersion,
			Status:       models.CompositedDocumentStatusPending,
		}
		if err := tx.Create(&doc).Error; err != nil {
			return err
		}
		compositedID = doc.ID

		return tx.Create(&models.ProcessingJob{
			JobType:     models.JobTypeCompositePDF,
			ReferenceID: doc.ID,
			Status:      "queued",
		}).Error
	})
	if err != nil {
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

type compositeAnswerRow struct {
	AssignmentFileID uuid.UUID
	ExtractedInkPath *string
	HasDefinedRegion bool
	PageNumber       *int
	RegionX          *float64
	RegionY          *float64
	RegionWidth      *float64
	RegionHeight     *float64
	QuestionNumber   int
	QuestionID       uuid.UUID
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

	var doc models.CompositedDocument
	if err := h.DB.Select("id", "submission_id", "version").First(&doc, "id = ?", compositedID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "composited document not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up composited document")
	}

	var submission models.Submission
	if err := h.DB.Select("id", "assignment_id").First(&submission, "id = ?", doc.SubmissionID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up submission")
	}

	var answerRows []compositeAnswerRow
	if err := h.DB.Table("answer_regions").
		Select(`questions.assignment_file_id, answer_regions.extracted_ink_path, questions.has_defined_region,
		        questions.page_number, questions.region_x, questions.region_y, questions.region_width, questions.region_height,
		        questions.question_number, questions.id AS question_id`).
		Joins("JOIN questions ON questions.id = answer_regions.question_id").
		Where("answer_regions.submission_id = ? AND answer_regions.status = ?", doc.SubmissionID, models.AnswerRegionStatusDone).
		Order("questions.question_number ASC").
		Find(&answerRows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load answers")
	}

	answers := make([]compositeAnswer, len(answerRows))
	for i, r := range answerRows {
		answers[i] = compositeAnswer{
			AssignmentFileID: r.AssignmentFileID,
			HasDefinedRegion: r.HasDefinedRegion,
			PageNumber:       r.PageNumber,
			RegionX:          r.RegionX,
			RegionY:          r.RegionY,
			RegionWidth:      r.RegionWidth,
			RegionHeight:     r.RegionHeight,
			QuestionNumber:   r.QuestionNumber,
			QuestionID:       r.QuestionID,
		}
		if r.ExtractedInkPath != nil {
			answers[i].ExtractedInkPath = *r.ExtractedInkPath
		}
	}

	files := []compositeAssignmentFile{}
	if err := h.DB.Table("assignment_files").
		Select("id, file_path").
		Where("assignment_id = ?", submission.AssignmentID).
		Find(&files).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load assignment files")
	}

	return c.JSON(fiber.Map{
		"submission_id":    doc.SubmissionID,
		"version":          doc.Version,
		"answers":          answers,
		"assignment_files": files,
	})
}

func (h *InternalHandler) StartComposite(c *fiber.Ctx) error {
	compositedID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid composited document id")
	}

	if err := h.DB.Model(&models.CompositedDocument{}).
		Where("id = ?", compositedID).
		Update("status", models.CompositedDocumentStatusGenerating).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to start compositing job")
	}
	if err := h.DB.Model(&models.ProcessingJob{}).
		Where("job_type = ? AND reference_id = ?", models.JobTypeCompositePDF, compositedID).
		Update("status", "running").Error; err != nil {
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

	var doc models.CompositedDocument
	if err := h.DB.Select("id", "submission_id").First(&doc, "id = ?", compositedID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "composited document not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up composited document")
	}
	submissionID := doc.SubmissionID

	switch req.Status {
	case "done":
		err := h.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.CompositedDocument{}).Where("id = ?", compositedID).Updates(map[string]any{
				"status":        models.CompositedDocumentStatusDone,
				"file_path":     req.FilePath,
				"error_message": nil,
			}).Error; err != nil {
				return err
			}
			for _, p := range req.Pages {
				if err := tx.Create(&models.CompositedDocumentPage{
					CompositedDocumentID: compositedID,
					PageNumber:           p.PageNumber,
					PageType:             models.CompositedPageType(p.PageType),
					QuestionID:           p.QuestionID,
				}).Error; err != nil {
					return err
				}
			}
			return tx.Model(&models.Submission{}).
				Where("id = ?", submissionID).
				Update("status", models.SubmissionStatusComposited).Error
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update composited document")
		}
		if err := h.DB.Model(&models.ProcessingJob{}).
			Where("job_type = ? AND reference_id = ?", models.JobTypeCompositePDF, compositedID).
			Update("status", "done").Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update job status")
		}
	case "failed":
		if err := h.DB.Model(&models.CompositedDocument{}).Where("id = ?", compositedID).Updates(map[string]any{
			"status":        models.CompositedDocumentStatusFailed,
			"error_message": req.ErrorMessage,
		}).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update composited document")
		}
		if err := h.DB.Model(&models.ProcessingJob{}).
			Where("job_type = ? AND reference_id = ?", models.JobTypeCompositePDF, compositedID).
			Updates(map[string]any{"status": "failed", "error_message": req.ErrorMessage}).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update job status")
		}
		if err := h.DB.Model(&models.Submission{}).
			Where("id = ?", submissionID).
			Update("status", models.SubmissionStatusFailed).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update submission")
		}
	default:
		return fiber.NewError(fiber.StatusBadRequest, "status must be 'done' or 'failed'")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
