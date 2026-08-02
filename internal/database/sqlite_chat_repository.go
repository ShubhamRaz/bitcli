// Package database owns SQLite connection setup, migrations, and concrete repositories.
package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/bitcli/bitcli/internal/utils"
)

// ChatSession is a saved local conversation.
type ChatSession struct {
	ID        string
	Title     string
	ModelID   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChatMessage is one persisted message in a local conversation.
type ChatMessage struct {
	ID         string
	SessionID  string
	Role       string
	Content    string
	TokenCount int
	CreatedAt  time.Time
}

// ChatRepository stores chat history locally.
type ChatRepository struct {
	db *DB
}

// NewChatRepository creates a SQLite-backed chat history repository.
func NewChatRepository(db *DB) *ChatRepository {
	return &ChatRepository{db: db}
}

// CreateSession creates a new conversation session.
func (r *ChatRepository) CreateSession(ctx context.Context, title, modelID string) (ChatSession, error) {
	now := time.Now().UTC()
	session := ChatSession{
		ID:        utils.NewID("chat"),
		Title:     title,
		ModelID:   modelID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := r.db.SQL.ExecContext(ctx, `INSERT INTO chat_sessions(id, title, model_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, session.ID, session.Title, session.ModelID,
		session.CreatedAt.Format(time.RFC3339Nano), session.UpdatedAt.Format(time.RFC3339Nano))
	return session, err
}

// ListSessions lists recent conversation sessions.
func (r *ChatRepository) ListSessions(ctx context.Context) ([]ChatSession, error) {
	rows, err := r.db.SQL.QueryContext(ctx, `SELECT id, title, model_id, created_at, updated_at
		FROM chat_sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ChatSession
	for rows.Next() {
		var s ChatSession
		var created, updated string
		if err := rows.Scan(&s.ID, &s.Title, &s.ModelID, &created, &updated); err != nil {
			return nil, err
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		s.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// AddMessage appends one message to a session.
func (r *ChatRepository) AddMessage(ctx context.Context, sessionID, role, content string, tokens int) (ChatMessage, error) {
	now := time.Now().UTC()
	msg := ChatMessage{
		ID:         utils.NewID("msg"),
		SessionID:  sessionID,
		Role:       role,
		Content:    content,
		TokenCount: tokens,
		CreatedAt:  now,
	}
	err := r.db.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO chat_messages(id, session_id, role, content, token_count, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, msg.ID, msg.SessionID, msg.Role, msg.Content, msg.TokenCount,
			msg.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE chat_sessions SET updated_at = ? WHERE id = ?`,
			now.Format(time.RFC3339Nano), sessionID)
		return err
	})
	return msg, err
}

// Messages returns all messages for a session in chronological order.
func (r *ChatRepository) Messages(ctx context.Context, sessionID string) ([]ChatMessage, error) {
	rows, err := r.db.SQL.QueryContext(ctx, `SELECT id, session_id, role, content, token_count, created_at
		FROM chat_messages WHERE session_id = ? ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		var created string
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.TokenCount, &created); err != nil {
			return nil, err
		}
		msg.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// DeleteSession removes a chat session and its messages.
func (r *ChatRepository) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := r.db.SQL.ExecContext(ctx, `DELETE FROM chat_sessions WHERE id = ?`, sessionID)
	return err
}

