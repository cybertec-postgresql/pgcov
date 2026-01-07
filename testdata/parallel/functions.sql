-- Source file with functions
CREATE OR REPLACE FUNCTION add_numbers(a INT, b INT) RETURNS INT AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION multiply_numbers(a INT, b INT) RETURNS INT AS $$
BEGIN
    RETURN a * b;
END;
$$ LANGUAGE plpgsql;
