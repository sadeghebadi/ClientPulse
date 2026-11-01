package auth

import (
	"testing"
)

func TestHashPasswordAndVerify(t *testing.T) {
	// Use lightweight params for quick unit test execution
	testParams := &Argon2Params{
		Memory:      8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	password := "Correct-Horse-Battery-Staple-123"
	hash, err := HashPassword(password, testParams)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	// Verify correct password
	match, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("verification returned error: %v", err)
	}
	if !match {
		t.Fatal("expected password to match hash")
	}

	// Verify incorrect password
	match, err = VerifyPassword("Wrong-Password-That-Is-Long-Enough", hash)
	if err != nil {
		t.Fatalf("verification returned error for wrong password: %v", err)
	}
	if match {
		t.Fatal("expected wrong password to not match")
	}
}

func TestPasswordTooShort(t *testing.T) {
	_, err := HashPassword("short-pass", nil)
	if err != ErrPasswordTooShort {
		t.Fatalf("expected ErrPasswordTooShort, got: %v", err)
	}

	err = ValidatePasswordStrength("abc")
	if err != ErrPasswordTooShort {
		t.Fatalf("expected ErrPasswordTooShort, got: %v", err)
	}
}

func TestTokenGenerationAndHashing(t *testing.T) {
	token1, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	token2, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("failed to generate second token: %v", err)
	}

	if token1 == token2 {
		t.Fatal("tokens should be cryptographically unique")
	}

	pepper := "super-secret-pepper"
	hash1 := HashToken(token1, pepper)
	hash2 := HashToken(token2, pepper)

	if hash1 == hash2 {
		t.Fatal("hashes should differ for different tokens")
	}

	// Deterministic hash with same pepper
	hash1Again := HashToken(token1, pepper)
	if hash1 != hash1Again {
		t.Fatal("token hashing must be deterministic for the same token and pepper")
	}
}

