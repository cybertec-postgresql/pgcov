-- String manipulation functions for testing
CREATE OR REPLACE FUNCTION concat_strings(a TEXT, b TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN a || b;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION upper_string(s TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN UPPER(s);
END;
$$ LANGUAGE plpgsql;
