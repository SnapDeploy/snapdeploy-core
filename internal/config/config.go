package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Clerk    ClerkConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver   string
	DSN      string
	MaxConns int
	MinConns int
}

// ClerkConfig holds Clerk configuration
type ClerkConfig struct {
	PublishableKey string
	SecretKey      string
	JWKSURL        string
	Issuer         string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// .env file is optional, so we don't return error if it doesn't exist
		fmt.Println("No .env file found, using environment variables")
	}

	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 30),
			IdleTimeout:  getEnvAsInt("SERVER_IDLE_TIMEOUT", 120),
		},
		Database: DatabaseConfig{
			Driver:   getEnv("DB_DRIVER", "sqlite3"),
			DSN:      getEnv("DB_DSN", "./data/snapdeploy.db"),
			MaxConns: getEnvAsInt("DB_MAX_CONNS", 25),
			MinConns: getEnvAsInt("DB_MIN_CONNS", 5),
		},
		Clerk: ClerkConfig{
			PublishableKey: getEnv("CLERK_PUBLISHABLE_KEY", ""),
			SecretKey:      getEnv("CLERK_SECRET_KEY", ""),
			JWKSURL:        getEnv("CLERK_JWKS_URL", ""),
			Issuer:         getEnv("CLERK_ISSUER", ""),
		},
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Clerk.PublishableKey == "" {
		return fmt.Errorf("CLERK_PUBLISHABLE_KEY is required")
	}
	if c.Clerk.SecretKey == "" {
		return fmt.Errorf("CLERK_SECRET_KEY is required")
	}
	if c.Clerk.JWKSURL == "" {
		return fmt.Errorf("CLERK_JWKS_URL is required")
	}
	if c.Clerk.Issuer == "" {
		return fmt.Errorf("CLERK_ISSUER is required")
	}
	return nil
}

// GetServerAddress returns the server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvAsInt gets an environment variable as integer with a fallback value
func getEnvAsInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}

// getEnvAsBool gets an environment variable as boolean with a fallback value
func getEnvAsBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return fallback
}

// getEnvAsSlice gets an environment variable as slice with a fallback value
func getEnvAsSlice(key, separator string, fallback []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, separator)
	}
	return fallback
}
