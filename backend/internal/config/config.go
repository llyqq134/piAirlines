package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr string
	DBDSN    string

	JWTIssuer      string
	JWTSecret      string
	JWTTTL         time.Duration
	RefreshTTLDays int
	FrontendOrigin string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

func Load() Config {
	ttlMin, _ := strconv.Atoi(getenv("JWT_TTL_MINUTES", "120"))
	refreshDays, _ := strconv.Atoi(getenv("REFRESH_TTL_DAYS", "14"))
	return Config{
		HTTPAddr:           getenv("HTTP_ADDR", "0.0.0.0:8080"),
		DBDSN:              getenv("DB_DSN", ""),
		JWTIssuer:          getenv("JWT_ISSUER", "airtickets"),
		JWTSecret:          getenv("JWT_SECRET", ""),
		JWTTTL:             time.Duration(ttlMin) * time.Minute,
		RefreshTTLDays:     refreshDays,
		FrontendOrigin:     getenv("FRONTEND_ORIGIN", "http://localhost:5173"),
		GoogleClientID:     getenv("OAUTH_GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getenv("OAUTH_GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getenv("OAUTH_GOOGLE_REDIRECT_URL", ""),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
