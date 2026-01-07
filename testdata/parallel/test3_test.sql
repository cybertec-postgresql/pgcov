-- Test 3: Date operations
CREATE TABLE test3_dates (id INT, created_at TIMESTAMP);
INSERT INTO test3_dates VALUES (1, NOW());
SELECT * FROM test3_dates WHERE created_at <= NOW();
DROP TABLE test3_dates;
