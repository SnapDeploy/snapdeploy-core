package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"

	_ "github.com/lib/pq"
)

// PostgresManager handles creation and deletion of user project databases
type PostgresManager struct {
	masterDB    *sql.DB
	connBaseURL string // Base URL without database name for constructing per-project URLs
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

	// Build base URL (without database name) for constructing per-project URLs
	// Format: postgresql://user:password@host:port
	connBaseURL := fmt.Sprintf("%s://%s@%s",
		parsedURL.Scheme,
		parsedURL.User.String(),
		parsedURL.Host,
	)

	return &PostgresManager{
		masterDB:    db,
		connBaseURL: connBaseURL,
	}, nil
}

// CreateDatabase creates a new database for a project
// If the database already exists, it will be dropped and recreated (fresh state)
func (m *PostgresManager) CreateDatabase(ctx context.Context, dbName string) error {
	log.Printf("[PostgresManager] Creating database: %s", dbName)

	// First, drop the database if it exists (we want fresh database on each deployment)
	if err := m.DropDatabase(ctx, dbName); err != nil {
		log.Printf("[PostgresManager] Warning: failed to drop existing database %s: %v", dbName, err)
		// Continue anyway - database might not exist
	}

	// Create the database
	query := fmt.Sprintf("CREATE DATABASE %s", dbName)
	_, err := m.masterDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}

	log.Printf("[PostgresManager] Successfully created database: %s", dbName)
	return nil
}

// DropDatabase drops a database
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
	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
	_, err = m.masterDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop database %s: %w", dbName, err)
	}

	log.Printf("[PostgresManager] Successfully dropped database: %s", dbName)
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

// GetDatabaseURL returns the connection string for a project database
func (m *PostgresManager) GetDatabaseURL(dbName string) string {
	return fmt.Sprintf("%s/%s?sslmode=require", m.connBaseURL, dbName)
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
