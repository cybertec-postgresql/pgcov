-- Test 4: JSON operations
CREATE TABLE test4_json (id INT, data JSONB);
INSERT INTO test4_json VALUES (1, '{"key": "value"}'::jsonb);
SELECT * FROM test4_json WHERE data->>'key' = 'value';
DROP TABLE test4_json;
