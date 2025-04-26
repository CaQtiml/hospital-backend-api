package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration values.
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	JWTSecret  string
	JWTExpiry  time.Duration
	ServerPort string
}

// Load loads configuration from environment variables or a .env file.
func Load() (*Config, error) {
	// Load .env file if it exists
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading environment variables directly")
	}

	jwtExpiryHoursStr := getEnv("JWT_EXPIRY_HOURS", "24") // Default to 24 hours
	jwtExpiryHours, err := strconv.Atoi(jwtExpiryHoursStr)
	if err != nil {
		log.Printf("Invalid JWT_EXPIRY_HOURS value: %s. Using default 24 hours.", jwtExpiryHoursStr)
		jwtExpiryHours = 24
	}

	cfg := &Config{
		DBHost:     getEnv("DB_HOST", "db"), // Default to docker-compose service name
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "hospital_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		JWTSecret:  getEnv("JWT_SECRET", "a_very_secret_key"),
		JWTExpiry:  time.Hour * time.Duration(jwtExpiryHours),
		ServerPort: getEnv("SERVER_PORT", "8080"), // Port the Go app listens on internally
	}

	// Basic validation
	if cfg.JWTSecret == "a_very_secret_key" {
		log.Println("WARNING: JWT_SECRET is set to the default insecure value. Set a strong secret in your environment.")
	}
	if cfg.DBPassword == "password" {
		log.Println("WARNING: DB_PASSWORD is set to a weak default value. Set a strong password in your environment.")
	}

	return cfg, nil
}

// Helper function to get environment variables or return a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
