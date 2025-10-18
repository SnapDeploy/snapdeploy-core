package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"snapdeploy-core/internal/config"
)

// DB wraps the database connection and provides methods for database operations
type DB struct {
	conn *sql.DB
}

// NewConnection creates a new database connection
func NewConnection(cfg *config.DatabaseConfig) (*DB, error) {
	conn, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(cfg.MaxConns)
	conn.SetMaxIdleConns(cfg.MinConns)
	conn.SetConnMaxLifetime(time.Hour)

	// Test the connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// GetConnection returns the underlying database connection
func (db *DB) GetConnection() *sql.DB {
	return db.conn
}

// Ping tests the database connection
func (db *DB) Ping() error {
	return db.conn.Ping()
}
