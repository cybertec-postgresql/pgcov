-- Trigger Definitions for Testing Coverage

-- Create a sample table for triggers
CREATE TABLE IF NOT EXISTS audit_log (
    id SERIAL PRIMARY KEY,
    table_name TEXT NOT NULL,
    operation TEXT NOT NULL,
    changed_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Trigger function to update timestamp
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to log changes
CREATE OR REPLACE FUNCTION log_change()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO audit_log (table_name, operation)
    VALUES (TG_TABLE_NAME, TG_OP);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
CREATE TRIGGER users_update_timestamp
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER users_log_insert
    AFTER INSERT ON users
    FOR EACH ROW
    EXECUTE FUNCTION log_change();
