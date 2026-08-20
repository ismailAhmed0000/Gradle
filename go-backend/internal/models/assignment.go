package models

import (
	"time"

	"github.com/google/uuid"
)

type Assignment struct {
	ID        uuid.UUID `json:"id"`
	OwnerID   uuid.UUID `json:"owner_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type AssignmentFile struct {
	ID           uuid.UUID `json:"id"`
	AssignmentID uuid.UUID `json:"assignment_id"`
	FilePath     string    `json:"file_path"`
	PageCount    int       `json:"page_count"`
	CreatedAt    time.Time `json:"created_at"`
	DownloadURL  string    `json:"download_url,omitempty"`
}

type Question struct {
	ID               uuid.UUID `json:"id"`
	AssignmentID     uuid.UUID `json:"assignment_id"`
	AssignmentFileID uuid.UUID `json:"assignment_file_id"`
	QuestionNumber   int       `json:"question_number"`
	HasDefinedRegion bool      `json:"has_defined_region"`
	PageNumber       *int      `json:"page_number,omitempty"`
	RegionX          *float64  `json:"region_x,omitempty"`
	RegionY          *float64  `json:"region_y,omitempty"`
	RegionWidth      *float64  `json:"region_width,omitempty"`
	RegionHeight     *float64  `json:"region_height,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type AssignmentDetail struct {
	Assignment
	Questions       []Question       `json:"questions"`
	AssignmentFiles []AssignmentFile `json:"assignment_files"`
}
