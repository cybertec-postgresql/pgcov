-- SQL-language functions for testing CTE-based coverage instrumentation (B6 fix)

-- Helper table for multi-statement SQL function (must precede function definition)
CREATE TABLE IF NOT EXISTS sqlfunc_log(
    id SERIAL PRIMARY KEY,
    value INT NOT NULL
);

-- Simple SQL function returning a scalar
CREATE OR REPLACE FUNCTION double_val(x INT)
RETURNS INT AS $$
    SELECT x * 2;
$$ LANGUAGE sql;

-- SQL function returning a computed value
CREATE OR REPLACE FUNCTION add_vals(a INT, b INT)
RETURNS INT AS $$
    SELECT a + b;
$$ LANGUAGE sql;

-- SQL function returning TEXT
CREATE OR REPLACE FUNCTION greet(name TEXT)
RETURNS TEXT AS $$
    SELECT 'Hello, ' || name || '!';
$$ LANGUAGE sql;

-- SQL function with multiple statements (last determines return type)
CREATE OR REPLACE FUNCTION insert_and_return(val INT)
RETURNS INT AS $$
    INSERT INTO sqlfunc_log(value) VALUES (val);
    SELECT val * 10;
$$ LANGUAGE sql;
