// Package database tests the SQLite chat history repository.
package database

import (
	"context"
	"testing"
)

func TestChatRepository_CreateAndListSessions(t *testing.T) {
	db := openTestDB(t)
	repo := NewChatRepository(db)
	ctx := context.Background()

	session, err := repo.CreateSession(ctx, "My Chat", "model-1")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID == "" {
		t.Fatal("session ID should not be empty")
	}
	if session.Title != "My Chat" {
		t.Fatalf("unexpected title: %q", session.Title)
	}

	sessions, err := repo.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != session.ID {
		t.Fatalf("listed session ID mismatch")
	}
}

func TestChatRepository_AddAndReadMessages(t *testing.T) {
	db := openTestDB(t)
	repo := NewChatRepository(db)
	ctx := context.Background()

	session, _ := repo.CreateSession(ctx, "Test", "model-1")

	msg, err := repo.AddMessage(ctx, session.ID, "user", "Hello!", 3)
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if msg.ID == "" {
		t.Fatal("message ID should not be empty")
	}

	_, err = repo.AddMessage(ctx, session.ID, "assistant", "Hi there!", 5)
	if err != nil {
		t.Fatalf("AddMessage (assistant): %v", err)
	}

	messages, err := repo.Messages(ctx, session.ID)
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Role != "user" {
		t.Fatalf("expected first message role 'user', got %q", messages[0].Role)
	}
	if messages[1].Role != "assistant" {
		t.Fatalf("expected second message role 'assistant', got %q", messages[1].Role)
	}
}

func TestChatRepository_DeleteSession(t *testing.T) {
	db := openTestDB(t)
	repo := NewChatRepository(db)
	ctx := context.Background()

	session, _ := repo.CreateSession(ctx, "To Delete", "model-1")
	_, _ = repo.AddMessage(ctx, session.ID, "user", "test", 1)

	if err := repo.DeleteSession(ctx, session.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	sessions, _ := repo.ListSessions(ctx)
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions after delete, got %d", len(sessions))
	}

	// Messages should be cascade-deleted.
	msgs, err := repo.Messages(ctx, session.ID)
	if err != nil {
		t.Fatalf("Messages after delete: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages after session delete (cascade), got %d", len(msgs))
	}
}

func TestChatRepository_MultipleSessionsOrdered(t *testing.T) {
	db := openTestDB(t)
	repo := NewChatRepository(db)
	ctx := context.Background()

	_, _ = repo.CreateSession(ctx, "First", "model-1")
	_, _ = repo.CreateSession(ctx, "Second", "model-1")

	sessions, err := repo.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}
