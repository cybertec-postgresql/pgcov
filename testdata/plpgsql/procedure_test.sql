-- Test file for stored procedures

-- Test log_event procedure
CALL log_event('test_event');
CALL log_event('another_event');

-- Test update_user_status procedure
CALL update_user_status(1, 'active');
CALL update_user_status(2, 'inactive');

-- Test batch_insert procedure
CALL batch_insert(1, 5);
