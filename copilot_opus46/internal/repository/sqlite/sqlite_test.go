package sqlite

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/model"
)

func tempDB(t *testing.T) *DB {
	t.Helper()
	f, err := os.CreateTemp("", "storage-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	db, err := New(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestUserRepository(t *testing.T) {
	db := tempDB(t)
	ctx := context.Background()

	// Create a user
	user := &model.User{
		ID:           "u_1",
		Username:     "admin",
		PasswordHash: "hash",
		CreatedAt:    time.Now().Truncate(time.Second),
	}
	if err := db.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}

	// Retrieve by username
	got, err := db.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != "u_1" || got.Username != "admin" {
		t.Errorf("unexpected user: %+v", got)
	}

	// Non-existent user
	got, err = db.GetByUsername(ctx, "nobody")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("expected nil for non-existent user")
	}
}

func TestSessionRepository(t *testing.T) {
	db := tempDB(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	session := &model.Session{
		ID:         "sess_1",
		UserID:     "u_1",
		ExpiresAt:  now.Add(time.Hour),
		CreatedAt:  now,
		LastSeenAt: now,
	}

	// Create
	if err := db.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}

	// Get
	got, err := db.GetSession(ctx, "sess_1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.UserID != "u_1" {
		t.Fatalf("unexpected session: %+v", got)
	}

	// Touch
	if err := db.Touch(ctx, "sess_1"); err != nil {
		t.Fatal(err)
	}

	// Delete
	if err := db.DeleteSession(ctx, "sess_1"); err != nil {
		t.Fatal(err)
	}
	got, err = db.GetSession(ctx, "sess_1")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestMetadataCache(t *testing.T) {
	db := tempDB(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Put directory cache
	entry := &model.DirectoryCacheEntry{
		Path:        "/photos",
		PayloadJSON: `{"entries":[]}`,
		FetchedAt:   now,
		ExpiresAt:   now.Add(time.Minute),
	}
	if err := db.PutDirectoryCache(ctx, entry); err != nil {
		t.Fatal(err)
	}

	// Get directory cache
	got, err := db.GetDirectoryCache(ctx, "/photos")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected cache entry, got nil")
	}
	if got.PayloadJSON != `{"entries":[]}` {
		t.Errorf("unexpected payload: %s", got.PayloadJSON)
	}

	// Invalidate prefix
	if err := db.InvalidatePrefix(ctx, "/photos"); err != nil {
		t.Fatal(err)
	}
	got, err = db.GetDirectoryCache(ctx, "/photos")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("expected nil after invalidation")
	}
}

func TestExpiredSession(t *testing.T) {
	db := tempDB(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	session := &model.Session{
		ID:         "sess_expired",
		UserID:     "u_1",
		ExpiresAt:  now.Add(-time.Hour), // already expired
		CreatedAt:  now.Add(-2 * time.Hour),
		LastSeenAt: now.Add(-time.Hour),
	}

	if err := db.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetSession(ctx, "sess_expired")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("expired session should return nil")
	}
}
