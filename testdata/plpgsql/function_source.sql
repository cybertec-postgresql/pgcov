-- PL/pgSQL Functions for Testing Coverage

-- Simple function
CREATE OR REPLACE FUNCTION add_numbers(a INT, b INT)
RETURNS INT AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;

-- Function with conditional logic
CREATE OR REPLACE FUNCTION get_grade(score INT)
RETURNS TEXT AS $$
BEGIN
    IF score >= 90 THEN
        RETURN 'A';
    ELSIF score >= 80 THEN
        RETURN 'B';
    ELSIF score >= 70 THEN
        RETURN 'C';
    ELSE
        RETURN 'F';
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function with loop
CREATE OR REPLACE FUNCTION sum_to_n(n INT)
RETURNS INT AS $$
DECLARE
    total INT := 0;
    i INT;
BEGIN
    FOR i IN 1..n LOOP
        total := total + i;
    END LOOP;
    RETURN total;
END;
$$ LANGUAGE plpgsql;

-- Function with exception handling
CREATE OR REPLACE FUNCTION safe_divide(a INT, b INT)
RETURNS NUMERIC AS $$
BEGIN
    RETURN a::NUMERIC / b;
EXCEPTION
    WHEN division_by_zero THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql;
