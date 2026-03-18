-- Demo Schema: Simple Product Inventory System
--
-- This file sets up the products table and seeds a few rows so the
-- functions in 02_functions.sql have data to work with.

CREATE TABLE products (
    id        SERIAL  PRIMARY KEY,
    name      TEXT    NOT NULL,
    price     NUMERIC NOT NULL,
    stock_qty INT     NOT NULL DEFAULT 0,
    category  TEXT    NOT NULL DEFAULT 'general'
);

-- Seed data: a variety of stock levels to exercise get_stock_status branches
INSERT INTO products (name, price, stock_qty, category) VALUES
    ('Laptop',     999.99, 25,  'electronics'),  -- in_stock
    ('Headphones',  49.99,  8,  'electronics'),  -- low_stock
    ('Desk Chair',  199.99, 0,  'furniture'),    -- out_of_stock
    ('Notebook',     4.99, 200, 'stationery');   -- in_stock
