package main

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Host string
	Port string

	// Storage settings
	MaxEntries int // Maximum number of entries to keep (0 = unlimited)
	EntryTTL   int // TTL in seconds (0 = never expire)
}

func LoadConfig() *Config {
	// Load .env file if exists (ignore error if not found)
	_ = godotenv.Load()

	return &Config{
		Host:       getEnv("HOST", "0.0.0.0"),
		Port:       getEnv("PORT", "8080"),
		MaxEntries: getIntEnv("MAX_ENTRIES", 1000),
		EntryTTL:   getIntEnv("ENTRY_TTL", 3600), // 1 hour default
	}
}

func (c *Config) Addr() string {
	return c.Host + ":" + c.Port
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
