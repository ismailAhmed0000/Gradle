package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gradle-go-backend/internal/auth"
	"gradle-go-backend/internal/config"
	"gradle-go-backend/internal/googleclassroom"
	"gradle-go-backend/internal/middleware"
	"gradle-go-backend/internal/models"
	"gradle-go-backend/internal/storage"
)

type GoogleIntegrationHandler struct {
	DB      *gorm.DB
	Config  *config.Config
	Storage *storage.S3Storage
}

func NewGoogleIntegrationHandler(db *gorm.DB, cfg *config.Config, s *storage.S3Storage) *GoogleIntegrationHandler {
	return &GoogleIntegrationHandler{DB: db, Config: cfg, Storage: s}
}

// TeacherAuthURL starts the teacher-side "connect Google Classroom" grant.
// The caller must already hold a Gradle session (dashboard) — the state
// carries their user id so /callback knows whose token to save.
func (h *GoogleIntegrationHandler) TeacherAuthURL(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}
	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		return fiber.NewError(fiber.StatusBadRequest, "redirect_uri is required")
	}

	oauthCfg, err := googleclassroom.OAuthConfig(h.Config, googleclassroom.FlowTeacherConnect)
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "google integration is not configured")
	}
	state := googleclassroom.EncodeState(h.Config.JWTSecret, googleclassroom.State{
		Flow:        googleclassroom.FlowTeacherConnect,
		UserID:      userID.String(),
		RedirectURI: redirectURI,
	})
	url := oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	return c.JSON(fiber.Map{"url": url})
}

// StudentAuthURL starts the "sign in with Google" flow for a student — no
// prior Gradle session exists yet, so redirect_uri must be an app deep link
// (or web page) that can pick a Gradle JWT up off the query string.
func (h *GoogleIntegrationHandler) StudentAuthURL(c *fiber.Ctx) error {
	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		return fiber.NewError(fiber.StatusBadRequest, "redirect_uri is required")
	}

	oauthCfg, err := googleclassroom.OAuthConfig(h.Config, googleclassroom.FlowStudentLogin)
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "google integration is not configured")
	}
	state := googleclassroom.EncodeState(h.Config.JWTSecret, googleclassroom.State{
		Flow:        googleclassroom.FlowStudentLogin,
		RedirectURI: redirectURI,
	})
	url := oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	return c.JSON(fiber.Map{"url": url})
}

// Callback handles Google's redirect for both flows, distinguished by the
// signed state param — there's a single registered redirect URI in the
// Google Cloud OAuth client either way.
func (h *GoogleIntegrationHandler) Callback(c *fiber.Ctx) error {
	if errParam := c.Query("error"); errParam != "" {
		return fiber.NewError(fiber.StatusBadRequest, "google authorization was denied")
	}

	state, err := googleclassroom.DecodeState(h.Config.JWTSecret, c.Query("state"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid or expired authorization state")
	}

	oauthCfg, err := googleclassroom.OAuthConfig(h.Config, state.Flow)
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "google integration is not configured")
	}

	ctx := c.Context()
	token, err := oauthCfg.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "failed to exchange authorization code")
	}
	if token.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "google did not grant offline access; please try connecting again")
	}

	email, _, err := fetchGoogleIdentity(ctx, oauthCfg.Client(context.Background(), token))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to read google account details")
	}

	switch state.Flow {
	case googleclassroom.FlowTeacherConnect:
		return h.finishTeacherConnect(c, state, token, email)
	case googleclassroom.FlowStudentLogin:
		return h.finishStudentLogin(c, state, token, email)
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unknown authorization flow")
	}
}

func fetchGoogleIdentity(ctx context.Context, client *http.Client) (email, sub string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return "", "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("userinfo request failed: %d", resp.StatusCode)
	}
	var body struct {
		Email string `json:"email"`
		Sub   string `json:"sub"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", err
	}
	return body.Email, body.Sub, nil
}

func (h *GoogleIntegrationHandler) finishTeacherConnect(c *fiber.Ctx, state googleclassroom.State, token *oauth2.Token, email string) error {
	userID, err := uuid.Parse(state.UserID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid authorization state")
	}
	cfg, _ := googleclassroom.OAuthConfig(h.Config, state.Flow)
	if err := googleclassroom.SaveToken(h.DB, h.Config, userID, token.RefreshToken, email, cfg.Scopes); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save google connection")
	}
	return c.Redirect(withQuery(state.RedirectURI, "connected", "1"))
}

func (h *GoogleIntegrationHandler) finishStudentLogin(c *fiber.Ctx, state googleclassroom.State, token *oauth2.Token, email string) error {
	if email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "google account has no email")
	}
	email = strings.ToLower(email)

	var user models.User
	err := h.DB.Where("email = ? AND role = ?", email, models.RoleStudent).First(&user).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		user = models.User{Email: email, Role: models.RoleStudent}
		if err := h.DB.Create(&user).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to create student account")
		}
	case err != nil:
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up account")
	}

	// Link this login to whichever roster row(s) a teacher's Classroom
	// import already created for this email but hasn't been claimed yet.
	if err := h.DB.Model(&models.Student{}).
		Where("email = ? AND user_id IS NULL", email).
		Update("user_id", user.ID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to link student roster record")
	}

	cfg, _ := googleclassroom.OAuthConfig(h.Config, state.Flow)
	if err := googleclassroom.SaveToken(h.DB, h.Config, user.ID, token.RefreshToken, email, cfg.Scopes); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save google connection")
	}

	jwtToken, err := auth.GenerateToken(user.ID, string(user.Role), h.Config.JWTSecret, h.Config.JWTExpiryHours)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to issue token")
	}
	return c.Redirect(withQuery(state.RedirectURI, "token", jwtToken))
}

func withQuery(rawURL, key, value string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}

func (h *GoogleIntegrationHandler) Status(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}
	connected, email, err := googleclassroom.IsConnected(h.DB, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up connection status")
	}
	return c.JSON(fiber.Map{"connected": connected, "google_email": email})
}

func (h *GoogleIntegrationHandler) Disconnect(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}
	if err := googleclassroom.Disconnect(h.DB, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to disconnect")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *GoogleIntegrationHandler) clientFor(c *fiber.Ctx, userID uuid.UUID, flow string) (*googleclassroom.Client, error) {
	httpClient, err := googleclassroom.HTTPClientFor(context.Background(), h.DB, h.Config, userID, flow)
	if err != nil {
		return nil, err
	}
	return googleclassroom.NewClient(context.Background(), httpClient)
}

func (h *GoogleIntegrationHandler) Courses(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}
	client, err := h.clientFor(c, userID, googleclassroom.FlowTeacherConnect)
	if err != nil {
		return googleConnectionError(err)
	}
	courses, err := client.ListTeachingCourses(context.Background())
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to list google classroom courses")
	}
	return c.JSON(courses)
}

type courseWorkResponse struct {
	googleclassroom.CourseWork
	AlreadyImported bool `json:"already_imported"`
}

func (h *GoogleIntegrationHandler) CourseWork(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}
	courseID := c.Params("id")

	client, err := h.clientFor(c, userID, googleclassroom.FlowTeacherConnect)
	if err != nil {
		return googleConnectionError(err)
	}
	work, err := client.ListCourseWork(context.Background(), courseID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to list coursework")
	}

	var importedIDs []string
	if err := h.DB.Model(&models.Assignment{}).
		Where("owner_id = ? AND external_course_id = ?", userID, courseID).
		Pluck("external_id", &importedIDs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to check imported assignments")
	}
	imported := make(map[string]bool, len(importedIDs))
	for _, id := range importedIDs {
		imported[id] = true
	}

	response := make([]courseWorkResponse, len(work))
	for i, w := range work {
		response[i] = courseWorkResponse{CourseWork: w, AlreadyImported: imported[w.ID]}
	}
	return c.JSON(response)
}

func googleConnectionError(err error) error {
	if errors.Is(err, googleclassroom.ErrNotConnected) {
		return fiber.NewError(fiber.StatusPreconditionFailed, "connect your google account first")
	}
	return fiber.NewError(fiber.StatusBadGateway, "failed to reach google")
}

type importRequest struct {
	CourseID      string    `json:"course_id"`
	SubjectID     uuid.UUID `json:"subject_id"`
	CourseWorkIDs []string  `json:"coursework_ids"`
}

type importResult struct {
	Imported []models.Assignment `json:"imported"`
}

// Import pulls the selected coursework (and any Drive-hosted PDF materials)
// into Gradle as read-only, source=classroom assignments, and syncs the
// course roster into Student rows so grading/enrollment work right away.
func (h *GoogleIntegrationHandler) Import(c *fiber.Ctx) error {
	ownerID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	var req importRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.CourseID == "" || req.SubjectID == uuid.Nil || len(req.CourseWorkIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "course_id, subject_id and coursework_ids are required")
	}

	var subject models.Subject
	if err := h.DB.Where("id = ? AND owner_id = ?", req.SubjectID, ownerID).First(&subject).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "subject not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to look up subject")
	}

	client, err := h.clientFor(c, ownerID, googleclassroom.FlowTeacherConnect)
	if err != nil {
		return googleConnectionError(err)
	}
	ctx := context.Background()

	if err := h.DB.Model(&models.Subject{}).
		Where("id = ? AND external_id IS NULL", subject.ID).
		Update("external_id", req.CourseID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to link subject to course")
	}

	if err := h.syncRoster(ctx, client, ownerID, req.CourseID, subject.ID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to sync class roster")
	}

	work, err := client.ListCourseWork(ctx, req.CourseID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to list coursework")
	}
	byID := make(map[string]googleclassroom.CourseWork, len(work))
	for _, w := range work {
		byID[w.ID] = w
	}

	imported := make([]models.Assignment, 0, len(req.CourseWorkIDs))
	for _, cwID := range req.CourseWorkIDs {
		cw, ok := byID[cwID]
		if !ok {
			continue
		}
		assignment, err := h.importCourseWork(ctx, client, ownerID, subject, cw)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("failed to import %q: %v", cw.Title, err))
		}
		imported = append(imported, assignment)
	}

	return c.Status(fiber.StatusCreated).JSON(importResult{Imported: imported})
}

func (h *GoogleIntegrationHandler) syncRoster(ctx context.Context, client *googleclassroom.Client, ownerID uuid.UUID, courseID string, subjectID uuid.UUID) error {
	roster, err := client.ListRoster(ctx, courseID)
	if err != nil {
		return err
	}
	for _, s := range roster {
		if s.Email == "" {
			continue
		}
		email := strings.ToLower(s.Email)
		student := models.Student{
			OwnerID:    ownerID,
			Name:       s.Name,
			Email:      &email,
			ExternalID: &s.UserID,
		}
		if err := h.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "owner_id"}, {Name: "external_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "email"}),
		}).Create(&student).Error; err != nil {
			return err
		}
		if err := h.DB.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&models.Enrollment{StudentID: student.ID, SubjectID: subjectID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h *GoogleIntegrationHandler) importCourseWork(
	ctx context.Context, client *googleclassroom.Client, ownerID uuid.UUID, subject models.Subject, cw googleclassroom.CourseWork,
) (models.Assignment, error) {
	var dueDate *time.Time
	if cw.DueDate != nil {
		parsed, err := time.Parse(time.RFC3339, *cw.DueDate)
		if err == nil {
			dueDate = &parsed
		}
	}

	assignment := models.Assignment{
		OwnerID:          ownerID,
		Title:            cw.Title,
		SubjectID:        &subject.ID,
		DueDate:          dueDate,
		Source:           models.AssignmentSourceClassroom,
		ExternalID:       &cw.ID,
		ExternalCourseID: &cw.CourseID,
	}
	err := h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "owner_id"}, {Name: "external_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "due_date"}),
	}).Create(&assignment).Error
	if err != nil {
		return models.Assignment{}, err
	}
	// OnConflict doesn't populate the id/created_at on the conflicting path
	// on every driver — re-read to be sure we return the real row.
	if err := h.DB.Where("owner_id = ? AND external_id = ?", ownerID, cw.ID).First(&assignment).Error; err != nil {
		return models.Assignment{}, err
	}

	for _, material := range cw.Materials {
		if material.DriveFileID == "" {
			continue
		}
		if err := h.importMaterial(ctx, client, assignment.ID, material); err != nil {
			return models.Assignment{}, err
		}
	}

	return assignment, nil
}

func (h *GoogleIntegrationHandler) importMaterial(
	ctx context.Context, client *googleclassroom.Client, assignmentID uuid.UUID, material googleclassroom.CourseWorkMaterial,
) error {
	data, contentType, err := client.DownloadDriveFile(ctx, material.DriveFileID)
	if err != nil {
		return err
	}
	// Only PDFs fit Gradle's paper-based grading pipeline (question regions
	// are defined against page geometry) — skip anything else silently
	// rather than failing the whole import over an unrelated attachment.
	if !strings.Contains(contentType, "pdf") && !strings.EqualFold(filepath.Ext(material.Title), ".pdf") {
		return nil
	}

	pageCount, err := api.PageCount(bytes.NewReader(data), model.NewDefaultConfiguration())
	if err != nil {
		return nil // not a valid PDF despite the content-type; skip it
	}

	assignmentFile := models.AssignmentFile{
		ID:           uuid.New(),
		AssignmentID: assignmentID,
		PageCount:    pageCount,
	}
	assignmentFile.FilePath = fmt.Sprintf("assignments/%s/%s.pdf", assignmentID, assignmentFile.ID)

	if err := h.Storage.PutObject(ctx, assignmentFile.FilePath, bytes.NewReader(data), int64(len(data)), "application/pdf"); err != nil {
		return err
	}
	return h.DB.Create(&assignmentFile).Error
}
