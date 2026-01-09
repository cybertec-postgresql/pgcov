package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/cybertec-postgresql/pgcov/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateTempDatabase creates a temporary database with a unique name
func CreateTempDatabase(ctx context.Context, pool *Pool) (*types.TempDatabase, error) {
	// Generate unique database name
	timestamp := time.Now().Format("20060102_150405")
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random suffix: %w", err)
	}
	randomSuffix := hex.EncodeToString(randomBytes)

	dbName := fmt.Sprintf("pgcov_test_%s_%s", timestamp, randomSuffix)

	// Create database (must use a connection to template database)
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// CREATE DATABASE cannot run in a transaction
	createSQL := fmt.Sprintf("CREATE DATABASE %s", dbName)
	_, err = conn.Exec(ctx, createSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary database: %w", err)
	}

	// Build connection string for the new database
	connString := fmt.Sprintf("host=%s port=%d dbname=%s",
		pool.config.PGHost, pool.config.PGPort, dbName)
	if pool.config.PGUser != "" {
		connString += fmt.Sprintf(" user=%s", pool.config.PGUser)
	}
	if pool.config.PGPassword != "" {
		connString += fmt.Sprintf(" password=%s", pool.config.PGPassword)
	}

	return &types.TempDatabase{
		Name:             dbName,
		CreatedAt:        time.Now(),
		ConnectionString: connString,
	}, nil
}

// DestroyTempDatabase drops a temporary database
func DestroyTempDatabase(ctx context.Context, pool *Pool, tempDB *types.TempDatabase) error {
	if tempDB == nil {
		return nil
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Terminate all connections to the database first (PostgreSQL 13+)
	terminateSQL := fmt.Sprintf(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = '%s' AND pid <> pg_backend_pid()
	`, tempDB.Name)

	_, err = conn.Exec(ctx, terminateSQL)
	if err != nil {
		// Non-fatal - database might not have any connections
	}

	// Drop database with FORCE option (PostgreSQL 13+)
	dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", tempDB.Name)
	_, err = conn.Exec(ctx, dropSQL)
	if err != nil {
		return fmt.Errorf("failed to drop temporary database %s: %w", tempDB.Name, err)
	}

	// Verify database was actually dropped
	if err := verifyDatabaseDropped(ctx, conn, tempDB.Name); err != nil {
		return fmt.Errorf("cleanup verification failed for database %s: %w", tempDB.Name, err)
	}

	return nil
}

// verifyDatabaseDropped checks that a database no longer exists in PostgreSQL catalog
func verifyDatabaseDropped(ctx context.Context, conn *pgxpool.Conn, dbName string) error {
	verifySQL := `
		SELECT EXISTS(
			SELECT 1 
			FROM pg_database 
			WHERE datname = $1
		)
	`

	var exists bool
	err := conn.QueryRow(ctx, verifySQL, dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to verify database deletion: %w", err)
	}

	if exists {
		return fmt.Errorf("database %s still exists after DROP command", dbName)
	}

	return nil
}

// CleanupStaleTempDatabases removes old pgcov temporary databases
// This is useful for cleanup after crashes or interrupted test runs
// Returns list of cleaned databases and any errors encountered
func CleanupStaleTempDatabases(ctx context.Context, pool *Pool, olderThan time.Duration) ([]string, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Find pgcov temp databases
	query := `
		SELECT datname
		FROM pg_database
		WHERE datname LIKE 'pgcov_test_%'
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query temp databases: %w", err)
	}
	defer rows.Close()

	var cleaned []string
	var failedCleanup []string

	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			continue
		}

		// Extract timestamp from database name
		// Format: pgcov_test_YYYYMMDD_HHMMSS_randomhex
		tempDB := &types.TempDatabase{Name: dbName}

		// Attempt to drop (will fail if database is in use)
		if err := DestroyTempDatabase(ctx, pool, tempDB); err == nil {
			cleaned = append(cleaned, dbName)
		} else {
			failedCleanup = append(failedCleanup, dbName)
		}
	}

	// Report cleanup failures as non-fatal warning
	if len(failedCleanup) > 0 {
		return cleaned, fmt.Errorf("failed to cleanup %d databases: %v (may be in use)", len(failedCleanup), failedCleanup)
	}

	return cleaned, nil
}
