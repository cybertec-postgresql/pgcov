-- Test 2: String operations
CREATE TABLE test2_strings (id INT, name TEXT);
INSERT INTO test2_strings VALUES (1, 'Alice'), (2, 'Bob');
SELECT * FROM test2_strings WHERE name LIKE 'A%';
DROP TABLE test2_strings;
