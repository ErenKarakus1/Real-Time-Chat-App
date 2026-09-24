package auth

import "testing"

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("secret-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if hash == "" {
		t.Fatal("expected password hash")
	}

	if hash == "secret-password" {
		t.Fatal("expected hash to differ from plain password")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if !CheckPassword("secret-password", hash) {
		t.Fatal("expected password to match hash")
	}

	if CheckPassword("wrong-password", hash) {
		t.Fatal("expected wrong password not to match hash")
	}
}
