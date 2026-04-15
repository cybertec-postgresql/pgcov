-- Tests for SQL-language functions
-- Verifies that CTE-based instrumentation does not break return types (B6)

DO $$
BEGIN
    -- Test double_val
    ASSERT double_val(5) = 10, 'double_val(5) should be 10';
    ASSERT double_val(0) = 0, 'double_val(0) should be 0';
    ASSERT double_val(-3) = -6, 'double_val(-3) should be -6';

    -- Test add_vals
    ASSERT add_vals(2, 3) = 5, 'add_vals(2,3) should be 5';
    ASSERT add_vals(0, 0) = 0, 'add_vals(0,0) should be 0';

    -- Test greet
    ASSERT greet('World') = 'Hello, World!', 'greet should format greeting';

    -- Test multi-statement SQL function
    ASSERT insert_and_return(7) = 70, 'insert_and_return(7) should be 70';
END $$;
