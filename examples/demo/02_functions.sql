-- Demo Functions: Product Management
--
-- Three PL/pgSQL functions designed to demonstrate pgcov branch coverage.
-- Key design: IF/ELSIF/ELSE branches use variable assignments (not RETURN)
-- so that coverage signals are *reachable* — they fire inside the preceding
-- branch body after the assignment, before ELSIF/ELSE takes over.

-- ---------------------------------------------------------------------------
-- add_product
--   Validates inputs then inserts a new product row.
--   Uses simple guard IFs — the NOTIFY fires before each IF check, so the
--   validation guards register as covered on every successful call.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION add_product(
    p_name     TEXT,
    p_price    NUMERIC,
    p_stock    INT     DEFAULT 0,
    p_category TEXT    DEFAULT 'general'
) RETURNS INT AS $$
DECLARE
    v_id INT;
BEGIN
    IF p_name IS NULL OR p_name = '' THEN
        RAISE EXCEPTION 'Product name cannot be empty';
    END IF;

    IF p_price < 0 THEN
        RAISE EXCEPTION 'Price must be non-negative, got %', p_price;
    END IF;

    IF p_stock < 0 THEN
        RAISE EXCEPTION 'Stock must be non-negative, got %', p_stock;
    END IF;

    INSERT INTO products (name, price, stock_qty, category)
    VALUES (p_name, p_price, p_stock, p_category)
    RETURNING id INTO v_id;

    RETURN v_id;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- get_stock_status
--   Returns a human-readable stock level for the given product.
--   Uses a result *variable* (not RETURN inside each branch) so that the
--   coverage signal for each ELSIF/ELSE clause fires when the *preceding*
--   branch executes — making uncovered branches clearly visible.
--
--   Branch signals:
--     'out_of_stock' signal  → only fires when stock = 0
--     'low_stock'   signal   → only fires when 0 < stock ≤ 10
--   (both are missed by the basic test that uses stock = 50)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION get_stock_status(
    p_product_id INT
) RETURNS TEXT AS $$
DECLARE
    v_stock  INT;
    v_result TEXT;
BEGIN
    SELECT stock_qty INTO v_stock
    FROM   products
    WHERE  id = p_product_id;

    IF NOT FOUND THEN
        RETURN 'unknown';
    END IF;

    IF v_stock = 0 THEN
        v_result := 'out_of_stock';
    ELSIF v_stock <= 10 THEN
        v_result := 'low_stock';
    ELSE
        v_result := 'in_stock';
    END IF;

    RETURN v_result;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- apply_pricing_policy
--   Returns the final price after applying a customer-tier discount.
--   Uses the same result-variable pattern: each ELSIF/ELSE signal fires
--   inside the *preceding* tier's branch, so only tiers you actually test
--   will register as covered.
--
--   Branch signals:
--     'premium'  signal  → only fires when tier = 'vip'    (initial test)
--     'standard' signal  → only fires when tier = 'premium' (needs comment-out)
--     'none'     signal  → only fires when tier = 'standard'(needs comment-out)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION apply_pricing_policy(
    p_product_id INT,
    p_tier       TEXT DEFAULT 'none'
) RETURNS NUMERIC AS $$
DECLARE
    v_price    NUMERIC;
    v_discount NUMERIC;
BEGIN
    SELECT price INTO v_price
    FROM   products
    WHERE  id = p_product_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Product % not found', p_product_id;
    END IF;

    IF p_tier = 'vip' THEN
        v_discount := 0.25;
    ELSIF p_tier = 'premium' THEN
        v_discount := 0.15;
    ELSIF p_tier = 'standard' THEN
        v_discount := 0.05;
    ELSE
        v_discount := 0.00;
    END IF;

    RETURN ROUND(v_price * (1 - v_discount), 2);
END;
$$ LANGUAGE plpgsql;
