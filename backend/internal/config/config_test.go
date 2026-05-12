package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("JWT_TTL_MINUTES", "")
	t.Setenv("REFRESH_TTL_DAYS", "")
	t.Setenv("JWT_ISSUER", "")
	t.Setenv("FRONTEND_ORIGIN", "")

	cfg := Load()
	if cfg.HTTPAddr != "0.0.0.0:8080" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.JWTTTL.Minutes() != 120 {
		t.Fatalf("JWTTTL = %v, want 120m", cfg.JWTTTL)
	}
	if cfg.RefreshTTLDays != 14 {
		t.Fatalf("RefreshTTLDays = %d, want 14", cfg.RefreshTTLDays)
	}
	if cfg.JWTIssuer != "airtickets" {
		t.Fatalf("JWTIssuer = %q", cfg.JWTIssuer)
	}
	if cfg.FrontendOrigin != "http://localhost:5173" {
		t.Fatalf("FrontendOrigin = %q", cfg.FrontendOrigin)
	}
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", "127.0.0.1:9999")
	t.Setenv("JWT_TTL_MINUTES", "15")
	t.Setenv("REFRESH_TTL_DAYS", "3")
	t.Setenv("JWT_ISSUER", "custom")
	t.Setenv("FRONTEND_ORIGIN", "https://app.example.com")

	cfg := Load()
	if cfg.HTTPAddr != "127.0.0.1:9999" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.JWTTTL.Minutes() != 15 {
		t.Fatalf("JWTTTL = %v, want 15m", cfg.JWTTTL)
	}
	if cfg.RefreshTTLDays != 3 {
		t.Fatalf("RefreshTTLDays = %d, want 3", cfg.RefreshTTLDays)
	}
	if cfg.JWTIssuer != "custom" {
		t.Fatalf("JWTIssuer = %q", cfg.JWTIssuer)
	}
	if cfg.FrontendOrigin != "https://app.example.com" {
		t.Fatalf("FrontendOrigin = %q", cfg.FrontendOrigin)
	}
}

