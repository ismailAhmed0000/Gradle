package models

import (
	"time"

	"github.com/google/uuid"
)

type SubmissionStatus string

const (
	SubmissionStatusPending    SubmissionStatus = "pending"
	SubmissionStatusProcessing SubmissionStatus = "processing"
	SubmissionStatusComposited SubmissionStatus = "composited"
	SubmissionStatusFailed     SubmissionStatus = "failed"
)

type AnswerRegionStatus string

const (
	AnswerRegionStatusPending    AnswerRegionStatus = "pending"
	AnswerRegionStatusProcessing AnswerRegionStatus = "processing"
	AnswerRegionStatusDone       AnswerRegionStatus = "done"
	AnswerRegionStatusFailed     AnswerRegionStatus = "failed"
)

type CompositedDocumentStatus string

const (
	CompositedDocumentStatusPending    CompositedDocumentStatus = "pending"
	CompositedDocumentStatusGenerating CompositedDocumentStatus = "generating"
	CompositedDocumentStatusDone       CompositedDocumentStatus = "done"
	CompositedDocumentStatusFailed     CompositedDocumentStatus = "failed"
)

type CompositedPageType string

const (
	CompositedPageTypeOriginalWithOverlay CompositedPageType = "original_with_overlay"
	CompositedPageTypeAppendedBlank       CompositedPageType = "appended_blank"
)

type JobType string

const (
	JobTypeExtractInk   JobType = "extract_ink"
	JobTypeCompositePDF JobType = "composite_pdf"
)

type Submission struct {
	ID           uuid.UUID        `json:"id"`
	AssignmentID uuid.UUID        `json:"assignment_id"`
	StudentName  string           `json:"student_name"`
	Status       SubmissionStatus `json:"status"`
	CreatedAt    time.Time        `json:"created_at"`
}

type SubmissionPage struct {
	ID           uuid.UUID `json:"id"`
	SubmissionID uuid.UUID `json:"submission_id"`
	PageNumber   int       `json:"page_number"`
	RawImagePath string    `json:"raw_image_path"`
	CreatedAt    time.Time `json:"created_at"`
}

type AnswerRegion struct {
	ID               uuid.UUID          `json:"id"`
	SubmissionID     uuid.UUID          `json:"submission_id"`
	QuestionID       uuid.UUID          `json:"question_id"`
	QuestionNumber   int                `json:"question_number"`
	SourcePageID     uuid.UUID          `json:"source_page_id"`
	Status           AnswerRegionStatus `json:"status"`
	CropX            float64            `json:"crop_x"`
	CropY            float64            `json:"crop_y"`
	CropWidth        float64            `json:"crop_width"`
	CropHeight       float64            `json:"crop_height"`
	ExtractedInkPath *string            `json:"extracted_ink_path,omitempty"`
	ErrorMessage     *string            `json:"error_message,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
}

type CompositedDocument struct {
	ID           uuid.UUID                `json:"id"`
	SubmissionID uuid.UUID                `json:"submission_id"`
	Version      int                      `json:"version"`
	Status       CompositedDocumentStatus `json:"status"`
	FilePath     *string                  `json:"file_path,omitempty"`
	ErrorMessage *string                  `json:"error_message,omitempty"`
	CreatedAt    time.Time                `json:"created_at"`
}

type CompositedDocumentPage struct {
	ID                   uuid.UUID          `json:"id"`
	CompositedDocumentID uuid.UUID          `json:"composited_document_id"`
	PageNumber           int                `json:"page_number"`
	PageType             CompositedPageType `json:"page_type"`
	QuestionID           *uuid.UUID         `json:"question_id,omitempty"`
	CreatedAt            time.Time          `json:"created_at"`
}

type SubmissionSummary struct {
	Submission
	PageCount          int `json:"page_count"`
	AnswerRegionsDone  int `json:"answer_regions_done"`
	AnswerRegionsTotal int `json:"answer_regions_total"`
}

type SubmissionDetail struct {
	Submission
	Pages         []SubmissionPage    `json:"pages"`
	AnswerRegions []AnswerRegion      `json:"answer_regions"`
	Composited    *CompositedDocument `json:"composited_document,omitempty"`
}
