// Package auth implements session-based authentication using the existing
// iam_session table. Sessions are stored as Go gob-encoded bytea in PostgreSQL.
//
// The session token is read from the cookie named "session" and looked up
// in the iam_session table. The account_id is extracted from the gob-encoded
// data field and used to identify the user.
package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"net/http"
	"time"
)

type contextKey string

const (
	// ContextKeyAccountID is the context key for the authenticated account ID.
	ContextKeyAccountID contextKey = "account_id"
	// ContextKeyProfileID is the context key for the profile ID (project-scoped).
	ContextKeyProfileID contextKey = "profile_id"

	sessionCookieName = "session"
)

// SessionData mirrors the Go gob-encoded structure stored in iam_session.data.
type SessionData struct {
	Deadline time.Time
	Values   map[string]interface{}
}

func init() {
	// Register types for gob decoding
	gob.Register(map[string]interface{}{})
	gob.Register(time.Time{})
}

// SessionStore reads sessions from the database.
type SessionStore struct {
	DB *sql.DB
}

// NewSessionStore creates a new session store.
func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{DB: db}
}

// Authenticate reads the session cookie, looks up the token in iam_session,
// decodes the gob data, and returns the account_id. If the session is invalid
// or expired, it returns an empty string.
func (s *SessionStore) Authenticate(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}

	token := cookie.Value

	var data []byte
	var expiry time.Time
	err = s.DB.QueryRowContext(r.Context(),
		"SELECT data, expiry FROM iam_session WHERE token = $1", token,
	).Scan(&data, &expiry)
	if err != nil {
		return ""
	}

	// Check expiry
	if time.Now().After(expiry) {
		return ""
	}

	// Decode gob
	var session SessionData
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&session); err != nil {
		return ""
	}

	// Extract account_id from session values
	accountID, ok := session.Values["account_id"]
	if !ok {
		return ""
	}

	accountIDStr, ok := accountID.(string)
	if !ok {
		return ""
	}

	return accountIDStr
}

// GetProfileForProject finds the iam_profile for the given account in a project.
func (s *SessionStore) GetProfileForProject(ctx context.Context, accountID, projectID string) (string, error) {
	var profileID string
	err := s.DB.QueryRowContext(ctx, `
		SELECT p.id FROM iam_profile p
		JOIN iam_ident i ON i.id = p.ident_id
		WHERE i.account_id = $1 AND p.project_id = $2
			AND p.active = true AND p.removed = false`,
		accountID, projectID,
	).Scan(&profileID)
	return profileID, err
}

// AccountIDFromContext returns the account ID from the request context.
func AccountIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyAccountID).(string)
	return v
}

// WithAccountID adds the account ID to the request context.
func WithAccountID(ctx context.Context, accountID string) context.Context {
	return context.WithValue(ctx, ContextKeyAccountID, accountID)
}
