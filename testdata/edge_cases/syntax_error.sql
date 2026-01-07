-- This file contains intentional syntax errors for testing error handling

-- Missing FROM clause
SELECT WHERE id = 1;

-- Unclosed parenthesis
SELECT * FROM users WHERE (id = 1;

-- Invalid CREATE statement
CREATE TABLE;

-- Incomplete INSERT
INSERT INTO users;
