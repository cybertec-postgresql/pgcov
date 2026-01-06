-- DDL: Create a table
CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    customer_type TEXT NOT NULL,
    discount_rate NUMERIC DEFAULT 0
);

-- DML: Insert test data
INSERT INTO customers (name, customer_type, discount_rate) 
VALUES ('Alice', 'VIP', 0.20);

INSERT INTO customers (name, customer_type, discount_rate)
VALUES ('Bob', 'REGULAR', 0.10);

-- DML: Update statement
UPDATE customers 
SET discount_rate = 0.25 
WHERE customer_type = 'VIP';

-- Complex function with multiple statements
CREATE OR REPLACE FUNCTION calculate_discount(price NUMERIC, customer_type TEXT)
RETURNS NUMERIC AS $$
DECLARE
    discount_rate NUMERIC;
    final_price NUMERIC;
BEGIN
    discount_rate := 0;
    
    IF customer_type = 'VIP' THEN
        discount_rate := 0.20;
    ELSIF customer_type = 'REGULAR' THEN
        discount_rate := 0.10;
    ELSE
        discount_rate := 0.05;
    END IF;
    
    final_price := price * (1 - discount_rate);
    RETURN final_price;
END;
$$ LANGUAGE plpgsql;

-- DML: Select to verify data
SELECT COUNT(*) FROM customers;
