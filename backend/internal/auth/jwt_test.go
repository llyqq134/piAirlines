package auth

import (
	"testing"
	"time"
)

func TestSignAndParseJWT_Success(t *testing.T) {
	t.Parallel()

	issuer := "airtickets-test"
	secret := "super-secret"
	token, err := SignJWT(issuer, secret, time.Hour, 42, "user@example.com", "customer")
	if err != nil {
		t.Fatalf("SignJWT() error = %v", err)
	}

	claims, err := ParseJWT(issuer, secret, token)
	if err != nil {
		t.Fatalf("ParseJWT() error = %v", err)
	}
	if claims.UserID != 42 {
		t.Fatalf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("Email = %q, want %q", claims.Email, "user@example.com")
	}
	if claims.Role != "customer" {
		t.Fatalf("Role = %q, want %q", claims.Role, "customer")
	}
}

func TestParseJWT_WrongSecret(t *testing.T) {
	t.Parallel()

	token, err := SignJWT("airtickets-test", "secret-1", time.Hour, 1, "a@b.c", "admin")
	if err != nil {
		t.Fatalf("SignJWT() error = %v", err)
	}

	_, err = ParseJWT("airtickets-test", "secret-2", token)
	if err == nil {
		t.Fatal("ParseJWT() expected error for wrong secret")
	}
}

