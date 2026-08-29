package models

import (
	"time"

	"github.com/google/uuid"
)

type Student struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID   uuid.UUID `json:"owner_id" gorm:"type:uuid"`
	Name      string    `json:"name"`
	Email     *string   `json:"email,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type StudentSummary struct {
	Student
	Subjects []Subject `json:"subjects" gorm:"-"`
}

// StudentSubmission describes one assignment in a subject the student is
// enrolled in, plus their submission for it if they've started one — a
// student can be listed here with no submission yet. It's an output DTO
// assembled by hand in the handler, never queried directly.
type StudentSubmission struct {
	SubmissionID       *uuid.UUID        `json:"submission_id,omitempty"`
	AssignmentID       uuid.UUID         `json:"assignment_id"`
	AssignmentTitle    string            `json:"assignment_title"`
	SubjectName        *string           `json:"subject_name,omitempty"`
	Status             *SubmissionStatus `json:"status,omitempty"`
	PageCount          int               `json:"page_count"`
	AnswerRegionsDone  int               `json:"answer_regions_done"`
	AnswerRegionsTotal int               `json:"answer_regions_total"`
	CreatedAt          *time.Time        `json:"created_at,omitempty"`
}

type StudentDetail struct {
	Student
	Subjects    []Subject           `json:"subjects" gorm:"-"`
	Submissions []StudentSubmission `json:"submissions" gorm:"-"`
}

type Enrollment struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StudentID uuid.UUID `json:"student_id" gorm:"type:uuid"`
	SubjectID uuid.UUID `json:"subject_id" gorm:"type:uuid"`
	CreatedAt time.Time `json:"created_at"`
}
