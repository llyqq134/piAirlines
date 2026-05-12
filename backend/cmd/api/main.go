package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"airtickets/internal/config"
	"airtickets/internal/db"
	httpapi "airtickets/internal/http"
)

func main() {
	cfg := config.Load()
	if cfg.DBDSN == "" {
		log.Fatal("DB_DSN is required")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	migrationsPath := getenv("MIGRATIONS_PATH", "/migrations")
	if _, err := os.Stat(migrationsPath); err != nil {
		// try relative for local runs
		migrationsPath = filepath.Join("..", "..", "db", "migrations")
	}

	log.Printf("running migrations from %s", migrationsPath)
	if err := db.MigrateUp(cfg.DBDSN, migrationsPath); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.NewPool(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	srv := httpapi.NewServer(cfg, pool)
	log.Printf("listening on %s", cfg.HTTPAddr)
	if err := srv.HTTP.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("http: %v", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
