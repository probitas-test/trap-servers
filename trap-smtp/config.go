package main

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Host     string
	WebPort  string
	SMTPPort string

	// Storage settings
	MaxEntries int // Maximum number of entries to keep (0 = unlimited)
	EntryTTL   int // TTL in seconds (0 = never expire)

	// SMTP settings
	SMTPDomain    string
	SMTPMaxSize   int // Maximum message size in bytes
	AllowInsecure bool
}

func LoadConfig() *Config {
	// Load .env file if exists (ignore error if not found)
	_ = godotenv.Load()

	return &Config{
		Host:          getEnv("HOST", "0.0.0.0"),
		WebPort:       getEnv("WEB_PORT", "8080"),
		SMTPPort:      getEnv("SMTP_PORT", "2525"),
		MaxEntries:    getIntEnv("MAX_ENTRIES", 1000),
		EntryTTL:      getIntEnv("ENTRY_TTL", 3600),
		SMTPDomain:    getEnv("SMTP_DOMAIN", "localhost"),
		SMTPMaxSize:   getIntEnv("SMTP_MAX_SIZE", 10*1024*1024), // 10MB default
		AllowInsecure: getBoolEnv("ALLOW_INSECURE", true),
	}
}

func (c *Config) WebAddr() string {
	return c.Host + ":" + c.WebPort
}

func (c *Config) SMTPAddr() string {
	return c.Host + ":" + c.SMTPPort
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}
