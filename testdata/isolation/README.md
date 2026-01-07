# Isolation Test Fixtures

This directory contains test fixtures that demonstrate database isolation is working correctly in pgcov.

## Purpose

These tests deliberately create conflicting state modifications to verify that:
1. Each test runs in its own temporary database
2. Tests do not interfere with each other
3. Tests can be run in any order with identical results
4. Destructive operations (DROP TABLE) are isolated

## Test Files

### `shared_state.sql` (Source)
Contains functions that depend on a `users` table:
- `get_user_count()` - Returns count of users
- `get_latest_user()` - Returns name of most recent user

### `test_a_test.sql`
- Creates `users` table with schema version 1 (id, name)
- Inserts 3 users: Alice, Bob, Charlie
- Tests functions work with this schema
- Verifies table has 2 columns

### `test_b_test.sql`
- Creates `users` table with schema version 2 (id, name, email)
- Inserts 2 users: David, Eve (with emails)
- Tests functions work with this schema
- Verifies table has 3 columns
- Tests email data access

### `test_c_test.sql`
- Creates `users` table, inserts Frank
- Drops the entire table
- Recreates `users` table with fresh data
- Inserts 4 users: Grace, Henry, Isabel, Jack
- Tests functions after destructive operation

## What Would Fail Without Isolation

Without proper database isolation:
- **test_a** and **test_b** would conflict trying to create the same `users` table
- **test_c** would break other tests by dropping the shared `users` table
- Tests would have different results depending on execution order
- Table schema conflicts would cause errors

## Expected Behavior

With proper isolation, all three tests should:
- ✅ Pass independently in any order
- ✅ Each see their own version of the `users` table
- ✅ Not affect each other's state
- ✅ Produce identical coverage results regardless of order
