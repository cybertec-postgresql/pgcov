package database

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cybertec-postgresql/pgcov/pkg/types"
)

// skipIfNoPostgres skips the test if PostgreSQL is not available
func skipIfNoPostgres(t *testing.T) *Pool {
	t.Helper()

	// Use environment variables if available, otherwise use defaults
	pgPort := 5432
	if portStr := os.Getenv("PGPORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			pgPort = p
		}
	}

	config := &types.Config{
		PGHost:     getEnv("PGHOST", "localhost"),
		PGPort:     pgPort,
		PGUser:     getEnv("PGUSER", "postgres"),
		PGPassword: getEnv("PGPASSWORD", ""),
		PGDatabase: getEnv("PGDATABASE", "postgres"),
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	return pool
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestCreateTempDatabase(t *testing.T) {
	pool := skipIfNoPostgres(t)
	defer pool.Close()

	ctx := context.Background()

	tempDB, err := CreateTempDatabase(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTempDatabase() error = %v", err)
	}

	if tempDB == nil {
		t.Fatal("CreateTempDatabase() returned nil")
	}

	// Verify database name format
	if !strings.HasPrefix(tempDB.Name, "pgcov_test_") {
		t.Errorf("CreateTempDatabase() name = %q, want prefix 'pgcov_test_'", tempDB.Name)
	}

	// Verify creation timestamp is recent
	if time.Since(tempDB.CreatedAt) > 5*time.Second {
		t.Errorf("CreateTempDatabase() CreatedAt = %v, want recent", tempDB.CreatedAt)
	}

	// Verify connection string contains database name
	if !strings.Contains(tempDB.ConnectionString, tempDB.Name) {
		t.Errorf("CreateTempDatabase() ConnectionString = %q doesn't contain database name", tempDB.ConnectionString)
	}

	// Cleanup
	if err := DestroyTempDatabase(ctx, pool, tempDB); err != nil {
		t.Errorf("DestroyTempDatabase() error = %v", err)
	}
}

func TestDestroyTempDatabase(t *testing.T) {
	pool := skipIfNoPostgres(t)
	defer pool.Close()

	ctx := context.Background()

	// Create a database to destroy
	tempDB, err := CreateTempDatabase(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTempDatabase() error = %v", err)
	}

	// Destroy it
	err = DestroyTempDatabase(ctx, pool, tempDB)
	if err != nil {
		t.Fatalf("DestroyTempDatabase() error = %v", err)
	}

	// Verify database no longer exists
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	defer conn.Release()

	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", tempDB.Name).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check database existence: %v", err)
	}

	if exists {
		t.Errorf("DestroyTempDatabase() database %q still exists", tempDB.Name)
	}
}

func TestDestroyTempDatabase_Nil(t *testing.T) {
	pool := skipIfNoPostgres(t)
	defer pool.Close()

	ctx := context.Background()

	// Should not error on nil input
	err := DestroyTempDatabase(ctx, pool, nil)
	if err != nil {
		t.Errorf("DestroyTempDatabase(nil) error = %v, want nil", err)
	}
}

func TestCreateTempDatabase_UniqueName(t *testing.T) {
	pool := skipIfNoPostgres(t)
	defer pool.Close()

	ctx := context.Background()

	// Create multiple databases
	var databases []*types.TempDatabase
	for i := 0; i < 3; i++ {
		tempDB, err := CreateTempDatabase(ctx, pool)
		if err != nil {
			t.Fatalf("CreateTempDatabase() error = %v", err)
		}
		databases = append(databases, tempDB)
	}

	// Verify all names are unique
	names := make(map[string]bool)
	for _, db := range databases {
		if names[db.Name] {
			t.Errorf("CreateTempDatabase() produced duplicate name %q", db.Name)
		}
		names[db.Name] = true
	}

	// Cleanup
	for _, db := range databases {
		if err := DestroyTempDatabase(ctx, pool, db); err != nil {
			t.Errorf("DestroyTempDatabase() error = %v", err)
		}
	}
}

func TestCreateTempDatabase_Concurrent(t *testing.T) {
	pool := skipIfNoPostgres(t)
	defer pool.Close()

	ctx := context.Background()

	// Create databases concurrently
	numDBs := 5
	results := make(chan *types.TempDatabase, numDBs)
	errors := make(chan error, numDBs)

	for i := 0; i < numDBs; i++ {
		go func() {
			db, err := CreateTempDatabase(ctx, pool)
			if err != nil {
				errors <- err
				return
			}
			results <- db
		}()
	}

	// Collect results
	var databases []*types.TempDatabase
	for i := 0; i < numDBs; i++ {
		select {
		case db := <-results:
			databases = append(databases, db)
		case err := <-errors:
			t.Errorf("Concurrent CreateTempDatabase() error = %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("CreateTempDatabase() timeout")
		}
	}

	// Verify all names are unique
	names := make(map[string]bool)
	for _, db := range databases {
		if names[db.Name] {
			t.Errorf("Concurrent CreateTempDatabase() produced duplicate name %q", db.Name)
		}
		names[db.Name] = true
	}

	// Cleanup
	for _, db := range databases {
		if err := DestroyTempDatabase(ctx, pool, db); err != nil {
			t.Errorf("DestroyTempDatabase() error = %v", err)
		}
	}
}

func TestCleanupStaleTempDatabases(t *testing.T) {
	pool := skipIfNoPostgres(t)
	defer pool.Close()

	ctx := context.Background()

	// Create some temp databases
	var databases []*types.TempDatabase
	for i := 0; i < 3; i++ {
		tempDB, err := CreateTempDatabase(ctx, pool)
		if err != nil {
			t.Fatalf("CreateTempDatabase() error = %v", err)
		}
		databases = append(databases, tempDB)
	}

	// Cleanup all stale databases (older than 0 seconds = all)
	cleaned, err := CleanupStaleTempDatabases(ctx, pool, 0)
	if err != nil {
		t.Logf("CleanupStaleTempDatabases() warning: %v", err)
	}

	// Should have cleaned at least the ones we created
	if len(cleaned) < len(databases) {
		t.Logf("CleanupStaleTempDatabases() cleaned %d databases, created %d", len(cleaned), len(databases))
	}

	// Verify databases are gone
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	defer conn.Release()

	for _, db := range databases {
		var exists bool
		err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", db.Name).Scan(&exists)
		if err != nil {
			t.Fatalf("failed to check database existence: %v", err)
		}

		if exists {
			t.Errorf("CleanupStaleTempDatabases() database %q still exists", db.Name)
		}
	}
}

func TestTempDatabase_Lifecycle(t *testing.T) {
	pool := skipIfNoPostgres(t)
	defer pool.Close()

	ctx := context.Background()

	// Create
	tempDB, err := CreateTempDatabase(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTempDatabase() error = %v", err)
	}

	// Verify exists
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	defer conn.Release()

	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", tempDB.Name).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check database existence: %v", err)
	}

	if !exists {
		t.Errorf("Database %q does not exist after creation", tempDB.Name)
	}

	// Destroy
	if err := DestroyTempDatabase(ctx, pool, tempDB); err != nil {
		t.Fatalf("DestroyTempDatabase() error = %v", err)
	}

	// Verify does not exist
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", tempDB.Name).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check database existence: %v", err)
	}

	if exists {
		t.Errorf("Database %q still exists after destruction", tempDB.Name)
	}
}
