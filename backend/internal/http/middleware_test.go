package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"airtickets/internal/auth"
	"airtickets/internal/config"

	"github.com/gin-gonic/gin"
)

func TestAuthRequired_ValidToken(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		JWTIssuer: "airtickets-test",
		JWTSecret: "test-secret",
	}
	token, err := auth.SignJWT(cfg.JWTIssuer, cfg.JWTSecret, time.Hour, 1, "u@test", "customer")
	if err != nil {
		t.Fatalf("SignJWT() error = %v", err)
	}

	r := gin.New()
	r.GET("/protected", AuthRequired(cfg), func(c *gin.Context) {
		claims := MustClaims(c)
		if claims == nil || claims.UserID != 1 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAuthRequired_MissingToken(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := config.Config{JWTIssuer: "x", JWTSecret: "y"}
	r := gin.New()
	r.GET("/protected", AuthRequired(cfg), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		JWTIssuer: "airtickets-test",
		JWTSecret: "test-secret",
	}
	token, err := auth.SignJWT(cfg.JWTIssuer, cfg.JWTSecret, time.Hour, 1, "u@test", "customer")
	if err != nil {
		t.Fatalf("SignJWT() error = %v", err)
	}

	r := gin.New()
	r.GET("/admin-only", AuthRequired(cfg), RequireRole("admin"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

