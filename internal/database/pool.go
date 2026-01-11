package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cybertec-postgresql/pgcov/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

const applicationName = "pgcov"

// ConnectionError represents PostgreSQL connection failure
type ConnectionError struct {
	Host       string
	Port       int
	Message    string
	Suggestion string
}

func (e *ConnectionError) Error() string {
	msg := fmt.Sprintf("failed to connect to %s:%d: %s", e.Host, e.Port, e.Message)
	if e.Suggestion != "" {
		msg += fmt.Sprintf("\nSuggestion: %s", e.Suggestion)
	}
	return msg
}

// NewConnectionError creates a new ConnectionError
func NewConnectionError(host string, port int, message string) *ConnectionError {
	return &ConnectionError{
		Host:    host,
		Port:    port,
		Message: message,
	}
}

// Pool wraps pgxpool.Pool with additional functionality
type Pool struct {
	*pgxpool.Pool
	config *types.Config
}

// NewPool creates a new connection pool to PostgreSQL
func NewPool(ctx context.Context, config *types.Config) (*Pool, error) {
	// Use connection string directly from config
	connString := config.ConnectionString

	// Configure pool
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, &ConnectionError{
			Message:    fmt.Sprintf("invalid connection configuration: %v", err),
			Suggestion: "Check your PostgreSQL connection string format. Use URI format (postgresql://user:pass@host:port/db) or key=value format (host=localhost port=5432 ...)",
		}
	}

	poolConfig.ConnConfig.RuntimeParams["application_name"] = applicationName

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
		return nil, &ConnectionError{
			Message:    fmt.Sprintf("failed to create connection pool: %v", err),
			Suggestion: "Verify PostgreSQL is running and accessible with the provided connection string",
		}
	}

	// Check PostgreSQL version
	var versionStr string
	err = pool.QueryRow(ctx, "SHOW server_version_num").Scan(&versionStr)
	if err != nil {
		pool.Close()
		return nil, &ConnectionError{
			Message: fmt.Sprintf("failed to query PostgreSQL version: %v", err),
		}
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		pool.Close()
		return nil, &ConnectionError{
			Message: fmt.Sprintf("failed to parse PostgreSQL version '%s': %v", versionStr, err),
		}
	}

	// PostgreSQL 13+ required (version 130000+)
	if version < 130000 {
		pool.Close()
		return nil, &ConnectionError{
			Message:    fmt.Sprintf("PostgreSQL version %d is not supported (need 13+)", version/10000),
			Suggestion: "Upgrade to PostgreSQL 13 or later",
		}
	}

	return &Pool{
		Pool:   pool,
		config: config,
	}, nil
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
