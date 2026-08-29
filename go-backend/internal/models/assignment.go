package models

import (
	"time"

	"github.com/google/uuid"
)

type AssignmentStatus string

const (
	AssignmentStatusPending   AssignmentStatus = "pending"
	AssignmentStatusExpired   AssignmentStatus = "expired"
	AssignmentStatusSubmitted AssignmentStatus = "submitted"
	AssignmentStatusGraded    AssignmentStatus = "graded"
)

const (
	AssignmentSourceManual    = "manual"
	AssignmentSourceClassroom = "classroom"
)

type Assignment struct {
	ID               uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID          uuid.UUID        `json:"owner_id" gorm:"type:uuid"`
	Owner            *User            `json:"-" gorm:"foreignKey:OwnerID"`
	Title            string           `json:"title"`
	SubjectID        *uuid.UUID       `json:"subject_id,omitempty" gorm:"type:uuid"`
	Subject          *Subject         `json:"-" gorm:"foreignKey:SubjectID"`
	SubjectName      *string          `json:"subject_name,omitempty" gorm:"-"`
	DueDate          *time.Time       `json:"due_date,omitempty"`
	Status           AssignmentStatus `json:"status" gorm:"-"`
	Source           string           `json:"source"`
	ExternalID       *string          `json:"external_id,omitempty"`
	ExternalCourseID *string          `json:"external_course_id,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
}

type AssignmentFile struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AssignmentID uuid.UUID `json:"assignment_id" gorm:"type:uuid"`
	FilePath     string    `json:"file_path"`
	PageCount    int       `json:"page_count"`
	CreatedAt    time.Time `json:"created_at"`
	DownloadURL  string    `json:"download_url,omitempty" gorm:"-"`
}

type Question struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AssignmentID     uuid.UUID `json:"assignment_id" gorm:"type:uuid"`
	AssignmentFileID uuid.UUID `json:"assignment_file_id" gorm:"type:uuid"`
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
	TeacherEmail    string           `json:"teacher_email"`
	Questions       []Question       `json:"questions"`
	AssignmentFiles []AssignmentFile `json:"assignment_files"`
}
