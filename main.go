package main

import (
	"log"
	"os"

	"github.com/Larguma/stuff/internal/config"
	dbstore "github.com/Larguma/stuff/internal/db"
	"github.com/Larguma/stuff/internal/search"
	"github.com/Larguma/stuff/internal/server"
)

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("upload dir setup failed: %v", err)
	}

	db, err := dbstore.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database open failed: %v", err)
	}

	if err := dbstore.Migrate(db); err != nil {
		log.Fatalf("database migrate failed: %v", err)
	}

	index := search.NewIndex(db)
	if err := index.EnsureTables(); err != nil {
		log.Fatalf("search table setup failed: %v", err)
	}
	if err := index.Rebuild(); err != nil {
		log.Printf("search rebuild failed: %v", err)
	}

	router := server.NewRouter(db, cfg, index)
	log.Printf("stuff %s listening on %s", config.Version, cfg.Addr)
	if err := router.Run(cfg.Addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
