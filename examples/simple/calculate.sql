-- Simple source file with a basic function
CREATE OR REPLACE FUNCTION calculate_total(quantity INT, price NUMERIC)
RETURNS NUMERIC AS $$
BEGIN
    IF quantity < 0 THEN
        RAISE EXCEPTION 'Quantity cannot be negative';
    END IF;
    
    IF price < 0 THEN
        RAISE EXCEPTION 'Price cannot be negative';
    END IF;
    
    RETURN quantity * price;
END;
$$ LANGUAGE plpgsql;
