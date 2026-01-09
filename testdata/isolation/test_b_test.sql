-- Test B: Creates users table with schema version 2
-- This test would conflict with test_a if not properly isolated
-- Both tests create the same table name but with different schemas

-- Create users table (version 2: id + name + email)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT
);

-- Insert test data with emails
INSERT INTO users (name, email) VALUES ('David', 'david@example.com');
INSERT INTO users (name, email) VALUES ('Eve', 'eve@example.com');

-- Test the get_user_count function
SELECT get_user_count() = 2 AS test_user_count;

-- Test the get_latest_user function
SELECT get_latest_user() = 'Eve' AS test_latest_user;

-- Verify table structure (version 2 has 3 columns)
SELECT (COUNT(*) = 3) AS test_table_structure
FROM information_schema.columns 
WHERE table_name = 'users';

-- Verify email data exists
SELECT (email = 'eve@example.com') AS test_email_data FROM users WHERE name = 'Eve';
