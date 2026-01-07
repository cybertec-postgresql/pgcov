-- Test file for PL/pgSQL functions

-- Test add_numbers function
DO $$
BEGIN
    ASSERT add_numbers(2, 3) = 5, 'add_numbers failed';
    ASSERT add_numbers(0, 0) = 0, 'add_numbers zero case failed';
    ASSERT add_numbers(-1, 1) = 0, 'add_numbers negative case failed';
END $$;

-- Test get_grade function
DO $$
BEGIN
    ASSERT get_grade(95) = 'A', 'get_grade A case failed';
    ASSERT get_grade(85) = 'B', 'get_grade B case failed';
    ASSERT get_grade(75) = 'C', 'get_grade C case failed';
    ASSERT get_grade(60) = 'F', 'get_grade F case failed';
END $$;

-- Test sum_to_n function
DO $$
BEGIN
    ASSERT sum_to_n(5) = 15, 'sum_to_n failed';
    ASSERT sum_to_n(1) = 1, 'sum_to_n base case failed';
    ASSERT sum_to_n(0) = 0, 'sum_to_n zero case failed';
END $$;

-- Test safe_divide function
DO $$
BEGIN
    ASSERT safe_divide(10, 2) = 5, 'safe_divide normal case failed';
    ASSERT safe_divide(10, 0) IS NULL, 'safe_divide by zero should return NULL';
END $$;
