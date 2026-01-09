// Package testutil provides shared test utilities and helpers for integration tests.
// This package contains helpers for setting up PostgreSQL test containers and other
// common test infrastructure used across the project.
package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// PostgresImage is the Docker image used for PostgreSQL test containers
	PostgresImage = "docker.io/postgres:16-alpine"

	// Default test database credentials
	TestDatabase = "testdb"
	TestUsername = "testuser"
	TestPassword = "testpass"
)

// SetupPostgresContainer starts a PostgreSQL container and returns a connection string and cleanup function
func SetupPostgresContainer(t *testing.T) (string, func()) {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		PostgresImage,
		postgres.WithDatabase(TestDatabase),
		postgres.WithUsername(TestUsername),
		postgres.WithPassword(TestPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection details
	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=prefer",
		host, port.Port(), TestUsername, TestPassword, TestDatabase)

	cleanup := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return connString, cleanup
}
