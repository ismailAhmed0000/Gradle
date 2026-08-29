// Package googleclassroom wires Gradle up to a user's Google account for two
// purposes: a teacher importing courses/coursework/roster and pushing grades
// back (read + grade-write scopes), and a student turning their own work in
// (a narrower write scope that only Google will honor from the student's own
// grant, never the teacher's).
package googleclassroom

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"gradle-go-backend/internal/config"
)

const (
	FlowTeacherConnect = "teacher_connect"
	FlowStudentLogin   = "student_login"

	stateTTL = 10 * time.Minute
)

var teacherScopes = []string{
	"openid",
	"email",
	"https://www.googleapis.com/auth/classroom.courses.readonly",
	"https://www.googleapis.com/auth/classroom.coursework.me.readonly",
	"https://www.googleapis.com/auth/classroom.rosters.readonly",
	"https://www.googleapis.com/auth/classroom.profile.emails",
	"https://www.googleapis.com/auth/classroom.coursework.students",
	"https://www.googleapis.com/auth/drive.readonly",
}

var studentScopes = []string{
	"openid",
	"email",
	"https://www.googleapis.com/auth/classroom.courses.readonly",
	"https://www.googleapis.com/auth/classroom.coursework.me",
	"https://www.googleapis.com/auth/drive.file",
}

func OAuthConfig(cfg *config.Config, flow string) (*oauth2.Config, error) {
	scopes := teacherScopes
	if flow == FlowStudentLogin {
		scopes = studentScopes
	}
	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" || cfg.GoogleRedirectURL == "" {
		return nil, fmt.Errorf("google oauth is not configured")
	}
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}, nil
}

// State is a signed, short-lived payload round-tripped through Google as the
// `state` param so /callback knows what to do without server-side session
// storage. UserID is set for teacher_connect (they're already logged into
// the dashboard when they click "connect"); it's empty for student_login,
// where signing in *is* the point of the flow.
type State struct {
	Flow        string    `json:"flow"`
	UserID      string    `json:"user_id,omitempty"`
	RedirectURI string    `json:"redirect_uri"`
	Expiry      time.Time `json:"expiry"`
}

func EncodeState(secret string, s State) string {
	s.Expiry = time.Now().Add(stateTTL)
	body, _ := json.Marshal(s)
	payload := base64.RawURLEncoding.EncodeToString(body)
	mac := signState(secret, payload)
	return payload + "." + mac
}

func DecodeState(secret, encoded string) (State, error) {
	var s State
	payload, mac, found := strings.Cut(encoded, ".")
	if !found {
		return s, fmt.Errorf("malformed state")
	}
	if !hmac.Equal([]byte(mac), []byte(signState(secret, payload))) {
		return s, fmt.Errorf("invalid state signature")
	}
	body, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(body, &s); err != nil {
		return s, err
	}
	if time.Now().After(s.Expiry) {
		return s, fmt.Errorf("state expired")
	}
	return s, nil
}

func signState(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
