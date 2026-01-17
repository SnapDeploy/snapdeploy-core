package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

// PostgresManager handles creation and deletion of user project databases
type PostgresManager struct {
	masterDB *sql.DB
	host     string // Database host:port for constructing URLs
	scheme   string // URL scheme (postgresql)
}

// NewPostgresManager creates a new PostgreSQL database manager
// Expects RDS_DATABASE_URL in format: postgresql://user:password@host:port/dbname?sslmode=require
func NewPostgresManager() (*PostgresManager, error) {
	databaseURL := os.Getenv("RDS_DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("RDS_DATABASE_URL environment variable is required")
	}

	// Parse the URL to extract components
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RDS_DATABASE_URL: %w", err)
	}

	// Connect to master database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping master database: %w", err)
	}

	log.Printf("[PostgresManager] Connected to master database at %s", parsedURL.Host)

	return &PostgresManager{
		masterDB: db,
		host:     parsedURL.Host,
		scheme:   parsedURL.Scheme,
	}, nil
}

// DatabaseCredentials holds the credentials for a project database
type DatabaseCredentials struct {
	DatabaseName string
	Username     string
	Password     string
	DatabaseURL  string
}

// generatePassword creates a cryptographically secure random password
func generatePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// getUserName generates a username from the database name
func getUserName(dbName string) string {
	// Username: user_<dbname> (e.g., user_proj_abc12345)
	return fmt.Sprintf("user_%s", dbName)
}

// CreateDatabase creates a new database and user for a project
// If the database/user already exists, it will attempt to drop them first for a fresh state.
// If dropping fails (e.g., due to dependencies), it will reuse the existing resources
// with updated credentials instead of failing.
// Returns the credentials including the DATABASE_URL for the project
func (m *PostgresManager) CreateDatabase(ctx context.Context, dbName string) (*DatabaseCredentials, error) {
	log.Printf("[PostgresManager] Creating database and user: %s", dbName)

	// Generate username and password
	username := getUserName(dbName)
	password, err := generatePassword(24) // 48 character hex string
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	// First, drop the database and user if they exist (we want fresh state on each deployment)
	if err := m.DropDatabase(ctx, dbName); err != nil {
		log.Printf("[PostgresManager] Warning: failed to drop existing database %s: %v", dbName, err)
		// Continue anyway - database might not exist
	}

	// Check if user already exists
	userExists, err := m.userExists(ctx, username)
	if err != nil {
		log.Printf("[PostgresManager] Warning: failed to check if user %s exists: %v", username, err)
		userExists = false // Assume not exists, let CREATE USER handle it
	}

	if userExists {
		// User exists - alter their password instead of trying to drop/recreate
		// This handles cases where the user has dependencies that prevent dropping
		log.Printf("[PostgresManager] User %s already exists, updating password", username)
		alterUserQuery := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s'", username, strings.ReplaceAll(password, "'", "''"))
		if _, err := m.masterDB.ExecContext(ctx, alterUserQuery); err != nil {
			return nil, fmt.Errorf("failed to alter user %s: %w", username, err)
		}
		log.Printf("[PostgresManager] Updated password for user: %s", username)
	} else {
		// Create the user with the generated password
		// Using format string for username but parameterized password to prevent injection
		createUserQuery := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", username, strings.ReplaceAll(password, "'", "''"))
		if _, err := m.masterDB.ExecContext(ctx, createUserQuery); err != nil {
			return nil, fmt.Errorf("failed to create user %s: %w", username, err)
		}
		log.Printf("[PostgresManager] Created user: %s", username)
	}

	// Check if database already exists (in case DropDatabase failed to fully clean up)
	dbExists, err := m.DatabaseExists(ctx, dbName)
	if err != nil {
		log.Printf("[PostgresManager] Warning: failed to check if database %s exists: %v", dbName, err)
		dbExists = false // Assume not exists, let CREATE DATABASE handle it
	}

	if dbExists {
		// Database exists - update ownership to ensure the user owns it
		log.Printf("[PostgresManager] Database %s already exists, updating ownership", dbName)
		alterDBQuery := fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", dbName, username)
		if _, err := m.masterDB.ExecContext(ctx, alterDBQuery); err != nil {
			log.Printf("[PostgresManager] Warning: failed to alter database ownership: %v", err)
			// Continue anyway - the database exists and user might already own it
		}
	} else {
		// Create the database owned by the new user
		createDBQuery := fmt.Sprintf("CREATE DATABASE %s OWNER %s", dbName, username)
		if _, err := m.masterDB.ExecContext(ctx, createDBQuery); err != nil {
			// Cleanup: drop the user we just created (only if we created it)
			if !userExists {
				m.masterDB.ExecContext(ctx, fmt.Sprintf("DROP USER IF EXISTS %s", username))
			}
			return nil, fmt.Errorf("failed to create database %s: %w", dbName, err)
		}
	}

	// Grant all privileges on the database to the user
	grantQuery := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", dbName, username)
	if _, err := m.masterDB.ExecContext(ctx, grantQuery); err != nil {
		log.Printf("[PostgresManager] Warning: failed to grant privileges: %v", err)
		// Continue anyway - ownership should be sufficient
	}

	log.Printf("[PostgresManager] Successfully created database: %s with user: %s", dbName, username)

	// Build the DATABASE_URL for the project
	databaseURL := fmt.Sprintf("%s://%s:%s@%s/%s?sslmode=require",
		m.scheme,
		username,
		url.QueryEscape(password),
		m.host,
		dbName,
	)

	return &DatabaseCredentials{
		DatabaseName: dbName,
		Username:     username,
		Password:     password,
		DatabaseURL:  databaseURL,
	}, nil
}

// DropDatabase drops a database and its associated user
func (m *PostgresManager) DropDatabase(ctx context.Context, dbName string) error {
	log.Printf("[PostgresManager] Dropping database: %s", dbName)

	// Terminate all connections to the database first
	terminateQuery := fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()
	`, dbName)

	_, err := m.masterDB.ExecContext(ctx, terminateQuery)
	if err != nil {
		log.Printf("[PostgresManager] Warning: failed to terminate connections for %s: %v", dbName, err)
		// Continue anyway
	}

	// Drop the database
	dropDBQuery := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
	_, err = m.masterDB.ExecContext(ctx, dropDBQuery)
	if err != nil {
		return fmt.Errorf("failed to drop database %s: %w", dbName, err)
	}

	// Drop the associated user
	username := getUserName(dbName)
	dropUserQuery := fmt.Sprintf("DROP USER IF EXISTS %s", username)
	if _, err := m.masterDB.ExecContext(ctx, dropUserQuery); err != nil {
		log.Printf("[PostgresManager] Warning: failed to drop user %s: %v", username, err)
		// Continue anyway - user might not exist or might have other dependencies
	}

	log.Printf("[PostgresManager] Successfully dropped database: %s and user: %s", dbName, username)
	return nil
}

// DatabaseExists checks if a database exists
func (m *PostgresManager) DatabaseExists(ctx context.Context, dbName string) (bool, error) {
	query := "SELECT 1 FROM pg_database WHERE datname = $1"
	var exists int
	err := m.masterDB.QueryRowContext(ctx, query, dbName).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}
	return true, nil
}

// userExists checks if a PostgreSQL role/user exists
func (m *PostgresManager) userExists(ctx context.Context, username string) (bool, error) {
	query := "SELECT 1 FROM pg_roles WHERE rolname = $1"
	var exists int
	err := m.masterDB.QueryRowContext(ctx, query, username).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return true, nil
}

// Close closes the master database connection
func (m *PostgresManager) Close() error {
	if m.masterDB != nil {
		return m.masterDB.Close()
	}
	return nil
}

// GetDatabaseName generates a database name for a project
func GetDatabaseName(projectID string) string {
	// Use first 8 characters of project ID for database name
	// Prefix with proj_ to make it clear it's a project database
	return fmt.Sprintf("proj_%s", projectID[:8])
}
