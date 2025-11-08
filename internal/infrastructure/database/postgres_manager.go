package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// PostgresManager handles creation and deletion of user project databases
type PostgresManager struct {
	masterDB *sql.DB
	host     string
	port     string
	user     string
	password string
}

// NewPostgresManager creates a new PostgreSQL database manager
func NewPostgresManager() (*PostgresManager, error) {
	host := os.Getenv("RDS_HOST")
	port := os.Getenv("RDS_PORT")
	user := os.Getenv("RDS_USER")
	password := os.Getenv("RDS_PASSWORD")
	masterDBName := os.Getenv("RDS_DATABASE")

	if host == "" || port == "" || user == "" || password == "" || masterDBName == "" {
		return nil, fmt.Errorf("missing required RDS environment variables")
	}

	// Connect to master database (postgres) to create/drop other databases
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		host, port, user, password, masterDBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping master database: %w", err)
	}

	log.Printf("[PostgresManager] Connected to master database at %s:%s", host, port)

	return &PostgresManager{
		masterDB: db,
		host:     host,
		port:     port,
		user:     user,
		password: password,
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
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=require",
		m.user, m.password, m.host, m.port, dbName)
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
