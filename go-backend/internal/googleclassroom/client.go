package googleclassroom

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"google.golang.org/api/classroom/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type Client struct {
	Classroom *classroom.Service
	Drive     *drive.Service
}

func NewClient(ctx context.Context, httpClient *http.Client) (*Client, error) {
	cls, err := classroom.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("classroom service: %w", err)
	}
	drv, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("drive service: %w", err)
	}
	return &Client{Classroom: cls, Drive: drv}, nil
}

type Course struct {
	ID   string
	Name string
}

// ListTeachingCourses lists active courses the connected account teaches.
func (c *Client) ListTeachingCourses(ctx context.Context) ([]Course, error) {
	var courses []Course
	call := c.Classroom.Courses.List().TeacherId("me").CourseStates("ACTIVE").Context(ctx)
	err := call.Pages(ctx, func(resp *classroom.ListCoursesResponse) error {
		for _, course := range resp.Courses {
			courses = append(courses, Course{ID: course.Id, Name: course.Name})
		}
		return nil
	})
	return courses, err
}

type CourseWorkMaterial struct {
	DriveFileID string
	Title       string
}

type CourseWork struct {
	ID          string
	CourseID    string
	Title       string
	Description string
	DueDate     *string // RFC3339, combined from Classroom's separate Date/TimeOfDay
	Materials   []CourseWorkMaterial
}

func (c *Client) ListCourseWork(ctx context.Context, courseID string) ([]CourseWork, error) {
	var work []CourseWork
	call := c.Classroom.Courses.CourseWork.List(courseID).CourseWorkStates("PUBLISHED").Context(ctx)
	err := call.Pages(ctx, func(resp *classroom.ListCourseWorkResponse) error {
		for _, cw := range resp.CourseWork {
			item := CourseWork{
				ID:          cw.Id,
				CourseID:    cw.CourseId,
				Title:       cw.Title,
				Description: cw.Description,
				DueDate:     formatDueDate(cw.DueDate, cw.DueTime),
			}
			for _, m := range cw.Materials {
				if m.DriveFile != nil && m.DriveFile.DriveFile != nil {
					item.Materials = append(item.Materials, CourseWorkMaterial{
						DriveFileID: m.DriveFile.DriveFile.Id,
						Title:       m.DriveFile.DriveFile.Title,
					})
				}
			}
			work = append(work, item)
		}
		return nil
	})
	return work, err
}

func formatDueDate(d *classroom.Date, t *classroom.TimeOfDay) *string {
	if d == nil {
		return nil
	}
	hour, minute := 23, 59
	if t != nil {
		hour, minute = int(t.Hours), int(t.Minutes)
	}
	s := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:00Z", d.Year, d.Month, d.Day, hour, minute)
	return &s
}

type RosterStudent struct {
	UserID string
	Name   string
	Email  string
}

func (c *Client) ListRoster(ctx context.Context, courseID string) ([]RosterStudent, error) {
	var roster []RosterStudent
	call := c.Classroom.Courses.Students.List(courseID).Context(ctx)
	err := call.Pages(ctx, func(resp *classroom.ListStudentsResponse) error {
		for _, s := range resp.Students {
			name, email := "", ""
			if s.Profile != nil {
				email = s.Profile.EmailAddress
				if s.Profile.Name != nil {
					name = s.Profile.Name.FullName
				}
			}
			roster = append(roster, RosterStudent{UserID: s.UserId, Name: name, Email: email})
		}
		return nil
	})
	return roster, err
}

// DownloadDriveFile fetches attachment bytes using the teacher's drive.readonly
// grant (assignment materials the teacher attached to their own coursework).
func (c *Client) DownloadDriveFile(ctx context.Context, fileID string) ([]byte, string, error) {
	resp, err := c.Drive.Files.Get(fileID).Context(ctx).Download()
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// UploadDriveFile creates a new file in the *connected user's* Drive (used
// with a student's drive.file grant, so Gradle can only ever see files it
// created, never the rest of their Drive).
func (c *Client) UploadDriveFile(ctx context.Context, name, contentType string, data []byte) (fileID string, err error) {
	file, err := c.Drive.Files.Create(&drive.File{Name: name}).
		Media(bytes.NewReader(data), googleapi.ContentType(contentType)).
		Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return file.Id, nil
}

// FindOwnSubmission looks up the connected student's own submission for a
// piece of coursework — Classroom keys submissions by an id Gradle doesn't
// otherwise know, so this has to run before ModifyAttachments/TurnIn.
func (c *Client) FindOwnSubmission(ctx context.Context, courseID, courseWorkID string) (*classroom.StudentSubmission, error) {
	resp, err := c.Classroom.Courses.CourseWork.StudentSubmissions.
		List(courseID, courseWorkID).UserId("me").Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	if len(resp.StudentSubmissions) == 0 {
		return nil, fmt.Errorf("no submission found for this coursework")
	}
	return resp.StudentSubmissions[0], nil
}

// TurnInWithAttachment attaches the given Drive file to the student's own
// submission and turns it in. Must be called with a client built from the
// student's own token — Google rejects this from a teacher's grant.
func (c *Client) TurnInWithAttachment(ctx context.Context, courseID, courseWorkID, submissionID, driveFileID string) error {
	_, err := c.Classroom.Courses.CourseWork.StudentSubmissions.
		ModifyAttachments(courseID, courseWorkID, submissionID, &classroom.ModifyAttachmentsRequest{
			AddAttachments: []*classroom.Attachment{{DriveFile: &classroom.DriveFile{Id: driveFileID}}},
		}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("attach submission file: %w", err)
	}
	_, err = c.Classroom.Courses.CourseWork.StudentSubmissions.
		TurnIn(courseID, courseWorkID, submissionID, &classroom.TurnInStudentSubmissionRequest{}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("turn in submission: %w", err)
	}
	return nil
}

// PatchGradeAndReturn sets the grade on a submission and returns it to the
// student. Must be called with a client built from the course teacher's
// token — grading is a teacher-only action in Classroom.
func (c *Client) PatchGradeAndReturn(ctx context.Context, courseID, courseWorkID, submissionID string, grade float64) error {
	_, err := c.Classroom.Courses.CourseWork.StudentSubmissions.
		Patch(courseID, courseWorkID, submissionID, &classroom.StudentSubmission{AssignedGrade: grade}).
		UpdateMask("assignedGrade").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("set grade: %w", err)
	}
	_, err = c.Classroom.Courses.CourseWork.StudentSubmissions.
		Return(courseID, courseWorkID, submissionID, &classroom.ReturnStudentSubmissionRequest{}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("return submission: %w", err)
	}
	return nil
}

// ParseGrade converts Gradle's free-text grade field to the numeric score
// Classroom requires. Non-numeric marks (e.g. "A-") can't be synced.
func ParseGrade(grade string) (float64, error) {
	return strconv.ParseFloat(grade, 64)
}
