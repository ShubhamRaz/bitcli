// Package integration tests the BitCLI chat history lifecycle end-to-end.
package integration

import (
	"context"
	"testing"

	"github.com/bitcli/bitcli/internal/database"
)

func TestChatHistory_FullLifecycle(t *testing.T) {
	db := openIntegrationDB(t)
	repo := database.NewChatRepository(db)
	ctx := context.Background()

	// 1. Create a session.
	session, err := repo.CreateSession(ctx, "Integration Chat", "test-model")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID == "" {
		t.Fatal("session ID must not be empty")
	}

	// 2. Add messages.
	_, err = repo.AddMessage(ctx, session.ID, "user", "Hello, world!", 3)
	if err != nil {
		t.Fatalf("AddMessage (user): %v", err)
	}
	_, err = repo.AddMessage(ctx, session.ID, "assistant", "Hi there! How can I help?", 9)
	if err != nil {
		t.Fatalf("AddMessage (assistant): %v", err)
	}

	// 3. Read messages back in order.
	messages, err := repo.Messages(ctx, session.ID)
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Role != "user" {
		t.Fatalf("first message role: got %q, want 'user'", messages[0].Role)
	}
	if messages[1].Role != "assistant" {
		t.Fatalf("second message role: got %q, want 'assistant'", messages[1].Role)
	}
	if messages[0].Content != "Hello, world!" {
		t.Fatalf("first message content mismatch: %q", messages[0].Content)
	}

	// 4. List sessions.
	sessions, err := repo.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	// 5. Delete the session; messages should cascade.
	if err := repo.DeleteSession(ctx, session.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	sessions, _ = repo.ListSessions(ctx)
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions after delete, got %d", len(sessions))
	}

	msgs, _ := repo.Messages(ctx, session.ID)
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages after session delete (cascade), got %d", len(msgs))
	}
}

func TestChatHistory_MultipleSessionsIndependent(t *testing.T) {
	db := openIntegrationDB(t)
	repo := database.NewChatRepository(db)
	ctx := context.Background()

	s1, _ := repo.CreateSession(ctx, "Session A", "model-a")
	s2, _ := repo.CreateSession(ctx, "Session B", "model-b")

	_, _ = repo.AddMessage(ctx, s1.ID, "user", "Msg in A", 4)
	_, _ = repo.AddMessage(ctx, s2.ID, "user", "Msg in B", 4)

	msgsA, _ := repo.Messages(ctx, s1.ID)
	msgsB, _ := repo.Messages(ctx, s2.ID)

	if len(msgsA) != 1 {
		t.Fatalf("session A: expected 1 message, got %d", len(msgsA))
	}
	if len(msgsB) != 1 {
		t.Fatalf("session B: expected 1 message, got %d", len(msgsB))
	}
	if msgsA[0].Content != "Msg in A" {
		t.Fatalf("message in session A: %q", msgsA[0].Content)
	}
	if msgsB[0].Content != "Msg in B" {
		t.Fatalf("message in session B: %q", msgsB[0].Content)
	}
}
