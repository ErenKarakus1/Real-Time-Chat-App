package middleware

import "testing"

func TestBearerToken(t *testing.T) {
	token := bearerToken("Bearer abc.def.ghi")
	if token != "abc.def.ghi" {
		t.Fatalf("expected token, got %q", token)
	}
}

func TestBearerTokenRejectsInvalidHeader(t *testing.T) {
	tests := []string{
		"",
		"abc.def.ghi",
		"Basic abc.def.ghi",
		"Bearer",
		"Bearer abc def",
	}

	for _, tt := range tests {
		if token := bearerToken(tt); token != "" {
			t.Fatalf("expected empty token for %q, got %q", tt, token)
		}
	}
}
