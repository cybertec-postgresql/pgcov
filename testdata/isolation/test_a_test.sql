-- Test A: Creates users table with schema version 1
-- This test would conflict with test_b if not properly isolated
-- Both tests create the same table name but with different schemas

-- Create users table (version 1: id + name)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

-- Insert test data
INSERT INTO users (name) VALUES ('Alice');
INSERT INTO users (name) VALUES ('Bob');
INSERT INTO users (name) VALUES ('Charlie');

-- Test the get_user_count function
SELECT get_user_count() = 3 AS test_user_count;

-- Test the get_latest_user function
SELECT get_latest_user() = 'Charlie' AS test_latest_user;

-- Verify table structure (version 1)
SELECT (COUNT(*) = 2) AS test_table_structure
FROM information_schema.columns 
WHERE table_name = 'users';
