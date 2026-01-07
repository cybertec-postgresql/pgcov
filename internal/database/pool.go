package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pashagolub/pgcov/internal/errors"
	"github.com/pashagolub/pgcov/pkg/types"
)

// Pool wraps pgxpool.Pool with additional functionality
type Pool struct {
	*pgxpool.Pool
	config *types.Config
}

// NewPool creates a new connection pool to PostgreSQL
func NewPool(ctx context.Context, config *types.Config) (*Pool, error) {
	// Build connection string
	connString := buildConnectionString(config)

	// Configure pool
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, &errors.ConnectionError{
			Host:       config.PGHost,
			Port:       config.PGPort,
			Message:    fmt.Sprintf("invalid connection configuration: %v", err),
			Suggestion: "Check your PostgreSQL connection settings (host, port, user, password)",
		}
	}

	// Set pool size based on parallelism
	if config.Parallelism > 1 {
		// Need at least 2 connections per parallel test (one for exec, one for LISTEN)
		poolConfig.MaxConns = int32(config.Parallelism * 2)
	} else {
		poolConfig.MaxConns = 4 // Default for sequential execution
	}

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, &errors.ConnectionError{
			Host:       config.PGHost,
			Port:       config.PGPort,
			Message:    fmt.Sprintf("failed to create connection pool: %v", err),
			Suggestion: "Verify PostgreSQL is running and accessible at the specified host and port",
		}
	}

	// Test connection
	conn, err := pool.Acquire(ctx)
	if err != nil {
		pool.Close()
		return nil, &errors.ConnectionError{
			Host:       config.PGHost,
			Port:       config.PGPort,
			Message:    fmt.Sprintf("failed to acquire connection: %v", err),
			Suggestion: "Check PostgreSQL credentials and network connectivity",
		}
	}

	// Check PostgreSQL version
	var versionStr string
	err = conn.QueryRow(ctx, "SHOW server_version_num").Scan(&versionStr)
	conn.Release() // Release immediately after use

	if err != nil {
		pool.Close()
		return nil, &errors.ConnectionError{
			Host:    config.PGHost,
			Port:    config.PGPort,
			Message: fmt.Sprintf("failed to query PostgreSQL version: %v", err),
		}
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		pool.Close()
		return nil, &errors.ConnectionError{
			Host:    config.PGHost,
			Port:    config.PGPort,
			Message: fmt.Sprintf("failed to parse PostgreSQL version '%s': %v", versionStr, err),
		}
	}

	// PostgreSQL 13+ required (version 130000+)
	if version < 130000 {
		pool.Close()
		return nil, &errors.ConnectionError{
			Host:       config.PGHost,
			Port:       config.PGPort,
			Message:    fmt.Sprintf("PostgreSQL version %d is not supported (need 13+)", version/10000),
			Suggestion: "Upgrade to PostgreSQL 13 or later",
		}
	}

	return &Pool{
		Pool:   pool,
		config: config,
	}, nil
}

// buildConnectionString constructs a PostgreSQL connection string from config
func buildConnectionString(config *types.Config) string {
	connStr := fmt.Sprintf("host=%s port=%d", config.PGHost, config.PGPort)

	if config.PGUser != "" {
		connStr += fmt.Sprintf(" user=%s", config.PGUser)
	}
	if config.PGPassword != "" {
		connStr += fmt.Sprintf(" password=%s", config.PGPassword)
	}
	if config.PGDatabase != "" {
		connStr += fmt.Sprintf(" dbname=%s", config.PGDatabase)
	}

	// Additional settings
	connStr += " sslmode=prefer"

	return connStr
}

// Config returns the configuration used by this pool
func (p *Pool) Config() *types.Config {
	return p.config
}

// Close closes the connection pool
func (p *Pool) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
