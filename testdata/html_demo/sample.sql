-- Sample SQL file for testing
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert some test data
INSERT INTO users (username, email) VALUES
    ('alice', 'alice@example.com'),
    ('bob', 'bob@example.com');

-- Query example
SELECT * FROM users WHERE id = 1;

-- Update example
UPDATE users SET email = 'newemail@example.com' WHERE username = 'alice';

-- Conditional logic
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE username = 'admin') THEN
        RAISE NOTICE 'Admin user exists';
    ELSE
        INSERT INTO users (username, email) VALUES ('admin', 'admin@example.com');
    END IF;
END;
$$;
