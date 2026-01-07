-- Test file for calculate.sql
-- Tests must be in the same directory as source files

DO $$
BEGIN
    -- Test normal case
    ASSERT calculate_total(5, 10.00) = 50.00, 'Normal calculation failed';
    
    -- Test zero quantity
    ASSERT calculate_total(0, 10.00) = 0, 'Zero quantity calculation failed';
    
    -- Test negative quantity (should raise exception)
    BEGIN
        PERFORM calculate_total(-1, 10.00);
        RAISE EXCEPTION 'Should have raised exception for negative quantity';
    EXCEPTION
        WHEN OTHERS THEN
            -- Expected
    END;
    
    -- Test negative price (should raise exception)
    BEGIN
        PERFORM calculate_total(5, -10.00);
        RAISE EXCEPTION 'Should have raised exception for negative price';
    EXCEPTION
        WHEN OTHERS THEN
            -- Expected
    END;
    
    RAISE NOTICE 'All tests passed';
END $$;
