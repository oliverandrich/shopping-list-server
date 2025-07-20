package config

import (
	"os"
	"strconv"
)

type Config struct {
	SMTPHost   string
	SMTPPort   int
	SMTPUser   string
	SMTPPass   string
	SMTPFrom   string
	JWTSecret  []byte
	ServerPort string
	DBPath     string
}

func Load() *Config {
	cfg := &Config{
		SMTPHost:   getEnvOrDefault("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:   getEnvAsIntOrDefault("SMTP_PORT", 587),
		SMTPUser:   os.Getenv("SMTP_USER"),
		SMTPPass:   os.Getenv("SMTP_PASS"),
		SMTPFrom:   os.Getenv("SMTP_FROM"),
		ServerPort: getEnvOrDefault("PORT", ":3000"),
		DBPath:     getEnvOrDefault("DB_PATH", "shopping.db"),
	}

	// JWT Secret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-this-in-production"
	}
	cfg.JWTSecret = []byte(jwtSecret)

	return cfg
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
