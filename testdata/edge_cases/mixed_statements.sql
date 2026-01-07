-- Mixed SQL statement types for testing
-- Tests handling of diverse SQL commands

-- DDL statements
CREATE TABLE mixed_test (id INT, value TEXT);
ALTER TABLE mixed_test ADD COLUMN created_at TIMESTAMP;
CREATE INDEX idx_mixed_value ON mixed_test(value);

-- DML statements
INSERT INTO mixed_test VALUES (1, 'first');
INSERT INTO mixed_test VALUES (2, 'second');
UPDATE mixed_test SET value = 'updated' WHERE id = 1;
DELETE FROM mixed_test WHERE id = 2;

-- Query statements
SELECT * FROM mixed_test;
SELECT id, value FROM mixed_test WHERE id = 1;
SELECT COUNT(*) FROM mixed_test;

-- Transaction control (tests may wrap these)
-- BEGIN;
-- INSERT INTO mixed_test VALUES (3, 'third');
-- COMMIT;

-- Cleanup
DROP INDEX IF EXISTS idx_mixed_value;
DROP TABLE IF EXISTS mixed_test;
