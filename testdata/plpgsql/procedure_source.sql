-- Stored Procedures for Testing Coverage

-- Simple procedure
CREATE OR REPLACE PROCEDURE log_event(event_name TEXT)
LANGUAGE plpgsql AS $$
BEGIN
    RAISE NOTICE 'Event: %', event_name;
END;
$$;

-- Procedure with table operations
CREATE OR REPLACE PROCEDURE update_user_status(user_id INT, new_status TEXT)
LANGUAGE plpgsql AS $$
BEGIN
    -- This is a simplified example
    RAISE NOTICE 'Would update user % to status %', user_id, new_status;
END;
$$;

-- Procedure with transaction control
CREATE OR REPLACE PROCEDURE batch_insert(start_id INT, end_id INT)
LANGUAGE plpgsql AS $$
DECLARE
    i INT;
BEGIN
    FOR i IN start_id..end_id LOOP
        RAISE NOTICE 'Processing ID: %', i;
    END LOOP;
END;
$$;
