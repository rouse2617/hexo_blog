// Package session provides session management functionality.
package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/opsgenius/backend/pkg/models"
)

// Store provides PostgreSQL-backed session persistence.
type Store struct {
	db *sqlx.DB
}

// NewStore creates a new PostgreSQL session store.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// ConnectPostgres creates a new PostgreSQL connection.
func ConnectPostgres(host string, port int, database, user, password string, maxConns int) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		host, port, database, user, password)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxConns / 2)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// InitSchema initializes the database schema.
func (s *Store) InitSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS sessions (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			title VARCHAR(500) NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			INDEX idx_user_id (user_id),
			INDEX idx_expires_at (expires_at)
		);

		CREATE TABLE IF NOT EXISTS messages (
			id VARCHAR(255) PRIMARY KEY,
			session_id VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL,
			content TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
			INDEX idx_session_id (session_id),
			INDEX idx_timestamp (timestamp)
		);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

// Save saves a session to the database.
func (s *Store) Save(session *Session) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert or update session record
	sessionRecord := models.SessionRecord{
		ID:        session.ID,
		UserID:    session.UserID,
		Title:     session.Title,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
		ExpiresAt: session.ExpiresAt,
	}

	query := `
		INSERT INTO sessions (id, user_id, title, created_at, updated_at, expires_at)
		VALUES (:id, :user_id, :title, :created_at, :updated_at, :expires_at)
		ON CONFLICT (id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			title = EXCLUDED.title,
			updated_at = EXCLUDED.updated_at,
			expires_at = EXCLUDED.expires_at
	`

	_, err = tx.NamedExec(query, sessionRecord)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Delete existing messages for this session
	_, err = tx.Exec("DELETE FROM messages WHERE session_id = $1", session.ID)
	if err != nil {
		return fmt.Errorf("failed to delete old messages: %w", err)
	}

	// Insert messages
	if len(session.Messages) > 0 {
		messageQuery := `
			INSERT INTO messages (id, session_id, role, content, timestamp)
			VALUES (:id, :session_id, :role, :content, :timestamp)
		`

		for _, msg := range session.Messages {
			messageRecord := models.MessageRecord{
				ID:        msg.ID,
				SessionID: session.ID,
				Role:      msg.Role,
				Content:   msg.Content,
				Timestamp: msg.Timestamp,
			}

			_, err = tx.NamedExec(messageQuery, messageRecord)
			if err != nil {
				return fmt.Errorf("failed to save message: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Get retrieves a session from the database.
func (s *Store) Get(sessionID string) (*Session, error) {
	var sessionRecord models.SessionRecord
	err := s.db.Get(&sessionRecord, "SELECT * FROM sessions WHERE id = $1", sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Get messages
	var messageRecords []models.MessageRecord
	err = s.db.Select(&messageRecords,
		"SELECT * FROM messages WHERE session_id = $1 ORDER BY timestamp ASC",
		sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Convert to session
	session := &Session{
		ID:        sessionRecord.ID,
		UserID:    sessionRecord.UserID,
		Title:     sessionRecord.Title,
		CreatedAt: sessionRecord.CreatedAt,
		UpdatedAt: sessionRecord.UpdatedAt,
		ExpiresAt: sessionRecord.ExpiresAt,
		Messages:  make([]Message, len(messageRecords)),
	}

	for i, msgRecord := range messageRecords {
		session.Messages[i] = Message{
			ID:        msgRecord.ID,
			Role:      msgRecord.Role,
			Content:   msgRecord.Content,
			Timestamp: msgRecord.Timestamp,
		}
	}

	return session, nil
}

// Update updates an existing session.
func (s *Store) Update(session *Session) error {
	return s.Save(session)
}

// Delete deletes a session from the database.
func (s *Store) Delete(sessionID string) error {
	result, err := s.db.Exec("DELETE FROM sessions WHERE id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return nil
}

// ListByUser retrieves sessions for a specific user.
func (s *Store) ListByUser(userID string, limit int) ([]*Session, error) {
	var sessionRecords []models.SessionRecord
	err := s.db.Select(&sessionRecords,
		"SELECT * FROM sessions WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2",
		userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessions := make([]*Session, len(sessionRecords))
	for i, record := range sessionRecords {
		// Get messages for each session
		var messageRecords []models.MessageRecord
		err = s.db.Select(&messageRecords,
			"SELECT * FROM messages WHERE session_id = $1 ORDER BY timestamp ASC",
			record.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get messages for session %s: %w", record.ID, err)
		}

		messages := make([]Message, len(messageRecords))
		for j, msgRecord := range messageRecords {
			messages[j] = Message{
				ID:        msgRecord.ID,
				Role:      msgRecord.Role,
				Content:   msgRecord.Content,
				Timestamp: msgRecord.Timestamp,
			}
		}

		sessions[i] = &Session{
			ID:        record.ID,
			UserID:    record.UserID,
			Title:     record.Title,
			CreatedAt: record.CreatedAt,
			UpdatedAt: record.UpdatedAt,
			ExpiresAt: record.ExpiresAt,
			Messages:  messages,
		}
	}

	return sessions, nil
}

// DeleteExpired deletes all expired sessions.
func (s *Store) DeleteExpired() (int64, error) {
	result, err := s.db.Exec("DELETE FROM sessions WHERE expires_at < $1", time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Helper function to convert session to JSON for debugging
func (s *Session) ToJSON() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
