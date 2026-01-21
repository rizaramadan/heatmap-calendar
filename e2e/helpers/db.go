// Package helpers provides narrowly-scoped utilities for E2E testing.
//
// The db helper provides direct PostgreSQL access for test setup and verification.
// It intentionally exposes only Query and Exec methods to enforce a minimal API.
package helpers

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBHelper provides direct PostgreSQL query capabilities for E2E tests.
//
// Use this helper to:
//   - Set up test data before tests run
//   - Verify database state after operations
//   - Clean up test data after tests complete
//
// DBHelper intentionally exposes only two methods (Query and Exec) to keep
// the E2E testing interface minimal and focused.
type DBHelper struct {
	pool *pgxpool.Pool
}

// NewDBHelper creates a new database helper wrapping the given connection pool.
//
// The pool should be configured to connect to the test database, not production.
// Callers are responsible for closing the pool when tests are complete.
func NewDBHelper(pool *pgxpool.Pool) *DBHelper {
	return &DBHelper{pool: pool}
}

// Query executes a SELECT query and returns the resulting rows.
//
// Use Query for read operations that return data:
//
//	rows, err := db.Query(ctx, "SELECT id, name FROM entities WHERE type = $1", "person")
//	if err != nil {
//	    return err
//	}
//	defer rows.Close()
//
//	for rows.Next() {
//	    var id, name string
//	    if err := rows.Scan(&id, &name); err != nil {
//	        return err
//	    }
//	    // process row...
//	}
//
// The caller is responsible for closing the returned Rows.
func (h *DBHelper) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return h.pool.Query(ctx, sql, args...)
}

// Exec executes a non-SELECT query (INSERT, UPDATE, DELETE, etc).
//
// Use Exec for write operations that modify data:
//
//	tag, err := db.Exec(ctx, "DELETE FROM entities WHERE id = $1", entityID)
//	if err != nil {
//	    return err
//	}
//	if tag.RowsAffected() != 1 {
//	    return fmt.Errorf("expected 1 row affected, got %d", tag.RowsAffected())
//	}
//
// The returned CommandTag contains information about the number of rows affected.
func (h *DBHelper) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return h.pool.Exec(ctx, sql, args...)
}
