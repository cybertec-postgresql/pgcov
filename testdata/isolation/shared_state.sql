-- Source file that creates a function using a table
-- This demonstrates that tests can create the same table independently

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
DECLARE
    count_val INTEGER;
BEGIN
    SELECT COUNT(*) INTO count_val FROM users;
    RETURN count_val;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_latest_user()
RETURNS TEXT AS $$
DECLARE
    username TEXT;
BEGIN
    SELECT name INTO username FROM users ORDER BY id DESC LIMIT 1;
    RETURN username;
END;
$$ LANGUAGE plpgsql;
