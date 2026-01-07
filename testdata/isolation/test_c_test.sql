-- Test C: Creates users table and then drops/recreates it
-- This demonstrates that destructive operations are isolated
-- Without isolation, this would break other tests

-- Create initial users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO users (name) VALUES ('Frank');

-- Verify initial state
SELECT get_user_count() = 1 AS test_initial_count;

-- Drop and recreate with different data
DROP TABLE users;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO users (name) VALUES ('Grace');
INSERT INTO users (name) VALUES ('Henry');
INSERT INTO users (name) VALUES ('Isabel');
INSERT INTO users (name) VALUES ('Jack');

-- Test after recreation
SELECT get_user_count() = 4 AS test_final_count;
SELECT get_latest_user() = 'Jack' AS test_latest_after_drop;
