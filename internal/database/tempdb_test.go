package database

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cybertec-postgresql/pgcov/internal/testutil"
	"github.com/cybertec-postgresql/pgcov/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// setupPostgresPool starts a PostgreSQL container and returns a Pool connected to it
func setupPostgresPool(t *testing.T) (*Pool, func()) {
	t.Helper()

	connString, cleanup := testutil.SetupPostgresContainer(t)

	config := &types.Config{
		ConnectionString: connString,
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to create pool: %v", err)
	}

	return pool, func() {
		pool.Close()
		cleanup()
	}
}

func TestCreateTempDatabase(t *testing.T) {
	pool, cleanup := setupPostgresPool(t)
	defer cleanup()

	ctx := context.Background()

	tempPool, err := CreateTempDatabase(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTempDatabase() error = %v", err)
	}

	if tempPool == nil {
		t.Fatal("CreateTempDatabase() returned nil")
	}

	dbName := tempPool.Config().ConnConfig.Database

	// Verify database name format
	if !strings.HasPrefix(dbName, "pgcov_test_") {
		t.Errorf("CreateTempDatabase() name = %q, want prefix 'pgcov_test_'", dbName)
	}

	// Cleanup
	if err := DestroyTempDatabase(ctx, pool, tempPool); err != nil {
		t.Errorf("DestroyTempDatabase() error = %v", err)
	}
}

func TestDestroyTempDatabase(t *testing.T) {
	pool, cleanup := setupPostgresPool(t)
	defer cleanup()

	ctx := context.Background()

	// Create a database to destroy
	tempPool, err := CreateTempDatabase(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTempDatabase() error = %v", err)
	}
	dbName := tempPool.Config().ConnConfig.Database

	// Destroy it
	err = DestroyTempDatabase(ctx, pool, tempPool)
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
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check database existence: %v", err)
	}

	if exists {
		t.Errorf("DestroyTempDatabase() database %q still exists", dbName)
	}
}

func TestDestroyTempDatabase_Nil(t *testing.T) {
	pool, cleanup := setupPostgresPool(t)
	defer cleanup()

	ctx := context.Background()

	// Should not error on nil input
	err := DestroyTempDatabase(ctx, pool, nil)
	if err != nil {
		t.Errorf("DestroyTempDatabase(nil) error = %v, want nil", err)
	}
}

func TestCreateTempDatabase_UniqueName(t *testing.T) {
	pool, cleanup := setupPostgresPool(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple databases
	var pools []*pgxpool.Pool
	for i := 0; i < 3; i++ {
		tempPool, err := CreateTempDatabase(ctx, pool)
		if err != nil {
			t.Fatalf("CreateTempDatabase() error = %v", err)
		}
		pools = append(pools, tempPool)
	}

	// Verify all names are unique
	names := make(map[string]bool)
	for _, p := range pools {
		name := p.Config().ConnConfig.Database
		if names[name] {
			t.Errorf("CreateTempDatabase() produced duplicate name %q", name)
		}
		names[name] = true
	}

	// Cleanup
	for _, p := range pools {
		if err := DestroyTempDatabase(ctx, pool, p); err != nil {
			t.Errorf("DestroyTempDatabase() error = %v", err)
		}
	}
}

func TestCreateTempDatabase_Concurrent(t *testing.T) {
	pool, cleanup := setupPostgresPool(t)
	defer cleanup()

	ctx := context.Background()

	// Create databases concurrently
	numDBs := 5
	results := make(chan *pgxpool.Pool, numDBs)
	errors := make(chan error, numDBs)

	for i := 0; i < numDBs; i++ {
		go func() {
			p, err := CreateTempDatabase(ctx, pool)
			if err != nil {
				errors <- err
				return
			}
			results <- p
		}()
	}

	// Collect results
	var pools []*pgxpool.Pool
	for i := 0; i < numDBs; i++ {
		select {
		case p := <-results:
			pools = append(pools, p)
		case err := <-errors:
			t.Errorf("Concurrent CreateTempDatabase() error = %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("CreateTempDatabase() timeout")
		}
	}

	// Verify all names are unique
	names := make(map[string]bool)
	for _, p := range pools {
		name := p.Config().ConnConfig.Database
		if names[name] {
			t.Errorf("Concurrent CreateTempDatabase() produced duplicate name %q", name)
		}
		names[name] = true
	}

	// Cleanup
	for _, p := range pools {
		if err := DestroyTempDatabase(ctx, pool, p); err != nil {
			t.Errorf("DestroyTempDatabase() error = %v", err)
		}
	}
}

func TestTempDatabase_Lifecycle(t *testing.T) {
	pool, cleanup := setupPostgresPool(t)
	defer cleanup()

	ctx := context.Background()

	// Create
	tempPool, err := CreateTempDatabase(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTempDatabase() error = %v", err)
	}
	dbName := tempPool.Config().ConnConfig.Database

	// Verify exists
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	defer conn.Release()

	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check database existence: %v", err)
	}

	if !exists {
		t.Errorf("Database %q does not exist after creation", dbName)
	}

	// Destroy
	if err := DestroyTempDatabase(ctx, pool, tempPool); err != nil {
		t.Fatalf("DestroyTempDatabase() error = %v", err)
	}

	// Verify does not exist
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check database existence: %v", err)
	}

	if exists {
		t.Errorf("Database %q still exists after destruction", dbName)
	}
}
