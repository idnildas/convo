package main

import (
	"log"

	"convo/internal/server"
	"convo/internal/config"
	"convo/internal/database"
)

func main() {
	cfg := config.Load()

	// Connect to DB (if DB not available, Connect will return an error)
	if err := database.Connect(cfg.DSN); err != nil {
		log.Fatalf("DB connect error: %v", err)
	}

	// Run migrations if file exists (RunMigrations is tolerant if file missing)
	if err := database.RunMigrations("migrations"); err != nil {
		log.Fatalf("migrations error: %v", err)
	}

	// Start server
	srv := server.NewServer(":8080", database.GetDB(), cfg.JWTSecret, cfg.JWTTTLHrs)
	if err := srv.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
