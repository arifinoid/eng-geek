package config

import "os"

// Config holds process-wide settings loaded from the environment.
type Config struct {
	Port        string
	DatabaseURL string
	StaticDir   string
}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	return Config{
		Port:        env("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		StaticDir:   env("STATIC_DIR", "../../web/out"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
