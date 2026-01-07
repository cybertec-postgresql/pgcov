-- Large SQL file with many statements to test performance
-- Generated for testing purposes

CREATE TABLE IF NOT EXISTS large_test_table (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Generate 100 INSERT statements
INSERT INTO large_test_table (name) VALUES ('Row 1');
INSERT INTO large_test_table (name) VALUES ('Row 2');
INSERT INTO large_test_table (name) VALUES ('Row 3');
INSERT INTO large_test_table (name) VALUES ('Row 4');
INSERT INTO large_test_table (name) VALUES ('Row 5');
INSERT INTO large_test_table (name) VALUES ('Row 6');
INSERT INTO large_test_table (name) VALUES ('Row 7');
INSERT INTO large_test_table (name) VALUES ('Row 8');
INSERT INTO large_test_table (name) VALUES ('Row 9');
INSERT INTO large_test_table (name) VALUES ('Row 10');
INSERT INTO large_test_table (name) VALUES ('Row 11');
INSERT INTO large_test_table (name) VALUES ('Row 12');
INSERT INTO large_test_table (name) VALUES ('Row 13');
INSERT INTO large_test_table (name) VALUES ('Row 14');
INSERT INTO large_test_table (name) VALUES ('Row 15');
INSERT INTO large_test_table (name) VALUES ('Row 16');
INSERT INTO large_test_table (name) VALUES ('Row 17');
INSERT INTO large_test_table (name) VALUES ('Row 18');
INSERT INTO large_test_table (name) VALUES ('Row 19');
INSERT INTO large_test_table (name) VALUES ('Row 20');
INSERT INTO large_test_table (name) VALUES ('Row 21');
INSERT INTO large_test_table (name) VALUES ('Row 22');
INSERT INTO large_test_table (name) VALUES ('Row 23');
INSERT INTO large_test_table (name) VALUES ('Row 24');
INSERT INTO large_test_table (name) VALUES ('Row 25');
INSERT INTO large_test_table (name) VALUES ('Row 26');
INSERT INTO large_test_table (name) VALUES ('Row 27');
INSERT INTO large_test_table (name) VALUES ('Row 28');
INSERT INTO large_test_table (name) VALUES ('Row 29');
INSERT INTO large_test_table (name) VALUES ('Row 30');
INSERT INTO large_test_table (name) VALUES ('Row 31');
INSERT INTO large_test_table (name) VALUES ('Row 32');
INSERT INTO large_test_table (name) VALUES ('Row 33');
INSERT INTO large_test_table (name) VALUES ('Row 34');
INSERT INTO large_test_table (name) VALUES ('Row 35');
INSERT INTO large_test_table (name) VALUES ('Row 36');
INSERT INTO large_test_table (name) VALUES ('Row 37');
INSERT INTO large_test_table (name) VALUES ('Row 38');
INSERT INTO large_test_table (name) VALUES ('Row 39');
INSERT INTO large_test_table (name) VALUES ('Row 40');
INSERT INTO large_test_table (name) VALUES ('Row 41');
INSERT INTO large_test_table (name) VALUES ('Row 42');
INSERT INTO large_test_table (name) VALUES ('Row 43');
INSERT INTO large_test_table (name) VALUES ('Row 44');
INSERT INTO large_test_table (name) VALUES ('Row 45');
INSERT INTO large_test_table (name) VALUES ('Row 46');
INSERT INTO large_test_table (name) VALUES ('Row 47');
INSERT INTO large_test_table (name) VALUES ('Row 48');
INSERT INTO large_test_table (name) VALUES ('Row 49');
INSERT INTO large_test_table (name) VALUES ('Row 50');

-- More operations
SELECT COUNT(*) FROM large_test_table;
UPDATE large_test_table SET name = 'Updated' WHERE id % 2 = 0;
DELETE FROM large_test_table WHERE id > 40;
SELECT * FROM large_test_table ORDER BY name LIMIT 10;
