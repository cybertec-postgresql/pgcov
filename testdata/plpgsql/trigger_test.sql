-- Test file for triggers

-- Test update timestamp trigger
INSERT INTO users (name) VALUES ('Alice');
SELECT pg_sleep(0.1);
UPDATE users SET name = 'Alice Updated' WHERE name = 'Alice';

-- Verify timestamp was updated
DO $$
DECLARE
    last_updated TIMESTAMP;
BEGIN
    SELECT updated_at INTO last_updated FROM users WHERE name = 'Alice Updated';
    ASSERT last_updated IS NOT NULL, 'Timestamp should be set';
END $$;

-- Test audit log trigger
INSERT INTO users (name) VALUES ('Bob');

-- Verify audit log entry
DO $$
DECLARE
    log_count INT;
BEGIN
    SELECT COUNT(*) INTO log_count FROM audit_log WHERE table_name = 'users' AND operation = 'INSERT';
    ASSERT log_count >= 1, 'Audit log should have INSERT entry';
END $$;
