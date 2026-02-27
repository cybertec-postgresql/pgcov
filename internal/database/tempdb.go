package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateTempDatabase creates a temporary database and returns a pool connected to it.
// The database name is accessible via pool.Config().ConnConfig.Database.
func CreateTempDatabase(ctx context.Context, adminPool *Pool) (*pgxpool.Pool, error) {
	timestamp := time.Now().Format("20060102_150405")
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random suffix: %w", err)
	}
	randomSuffix := hex.EncodeToString(randomBytes)
	dbName := fmt.Sprintf("pgcov_test_%s_%s", timestamp, randomSuffix)

	_, err := adminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary database: %w", err)
	}

	// Build connection string for the new database, preserving all original options (sslmode, etc.)
	config := adminPool.Pool.Config()
	config.ConnConfig.Database = dbName

	tempPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = adminPool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		return nil, fmt.Errorf("failed to connect to temp database: %w", err)
	}

	return tempPool, nil
}

// DestroyTempDatabase closes the temp pool and drops its underlying database.
func DestroyTempDatabase(ctx context.Context, adminPool *Pool, tempPool *pgxpool.Pool) error {
	if tempPool == nil {
		return nil
	}
	tempPool.Close()
	_, err := adminPool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", tempPool.Config().ConnConfig.Database))
	return err
}
