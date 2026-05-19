package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var Version = "dev"

type Config struct {
	DBPath        string
	UploadDir     string
	SessionSecret string
	Addr          string
}

func Load() Config {
	cfg := Config{
		DBPath:    getEnv("APP_DB_PATH", "data/app.db"),
		UploadDir: getEnv("APP_UPLOAD_DIR", "data/uploads"),
		Addr:      getEnv("APP_ADDR", ":8080"),
	}

	secret := os.Getenv("APP_SESSION_SECRET")
	if secret == "" {
		secretFile := getEnv("APP_SESSION_SECRET_FILE", "data/session_secret")
		if value, err := os.ReadFile(secretFile); err == nil {
			secret = strings.TrimSpace(string(value))
		}

		if secret == "" {
			buf := make([]byte, 32)
			if _, err := rand.Read(buf); err != nil {
				log.Fatalf("failed to generate session secret: %v", err)
			}
			secret = hex.EncodeToString(buf)
			if err := os.MkdirAll(filepath.Dir(secretFile), 0o755); err != nil {
				log.Fatalf("failed to create session secret directory: %v", err)
			}
			if err := os.WriteFile(secretFile, []byte(secret+"\n"), 0o600); err != nil {
				log.Fatalf("failed to persist session secret: %v", err)
			}
			log.Printf("APP_SESSION_SECRET not set; persisted generated value to %s", secretFile)
		}
	}
	cfg.SessionSecret = secret

	return cfg
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
