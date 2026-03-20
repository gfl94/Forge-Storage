package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	password := "my-secure-password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !VerifyPassword(hash, password) {
		t.Error("VerifyPassword() should return true for correct password")
	}

	if VerifyPassword(hash, "wrong-password") {
		t.Error("VerifyPassword() should return false for wrong password")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("GenerateSessionID() error = %v", err)
	}
	id2, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("GenerateSessionID() error = %v", err)
	}

	if len(id1) != 64 { // 32 bytes hex-encoded
		t.Errorf("session ID length = %d, want 64", len(id1))
	}
	if id1 == id2 {
		t.Error("two session IDs should not be equal")
	}
}

func TestGenerateID(t *testing.T) {
	id, err := GenerateID("u")
	if err != nil {
		t.Fatalf("GenerateID() error = %v", err)
	}

	if id[:2] != "u_" {
		t.Errorf("ID should start with 'u_', got %q", id[:2])
	}
}
