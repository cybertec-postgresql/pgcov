-- Demo Tests: Product Inventory System
--
-- Run with:
--   pgcov run -c "postgresql://user@host/postgres?sslmode=disable" ./examples/demo/
--
-- STEP 1: Run as-is to observe partial coverage (~80 %).
-- STEP 2: Uncomment the second DO block and re-run to reach 100 %.

-- Basic tests: happy-path only
--
-- Covers:
--   [x] add_product       - all 5 signals (guard IFs + INSERT + RETURN fire on every call)
--   [x] get_stock_status  - SELECT, guard IF, stock-level IF block start, RETURN
--   [ ] get_stock_status  - 'out_of_stock' and 'low_stock' branch signals NOT hit
--   [x] apply_pricing_policy - SELECT, guard IF, tier IF block start, 'premium' signal
--   [ ] apply_pricing_policy - 'standard' and 'none' branch signals NOT hit

DO $$
DECLARE
    v_id    INT;
    v_price NUMERIC;
BEGIN
    -- add_product: normal case
    v_id := add_product('Keyboard', 79.99, 50, 'electronics');
    ASSERT v_id IS NOT NULL, 'Expected a valid product ID';

    -- get_stock_status: in_stock branch (stock_qty = 50)
    ASSERT get_stock_status(v_id) = 'in_stock',
        format('Expected in_stock, got %s', get_stock_status(v_id));

    -- apply_pricing_policy: vip tier (25 % off)
    v_price := apply_pricing_policy(v_id, 'vip');
    ASSERT v_price < 79.99, format('Expected discounted price, got %s', v_price);

    RAISE NOTICE 'Basic tests passed -- run pgcov to see which branches are not covered!';
END $$;


-- Branch tests: uncomment to reach 100 % coverage
--
-- How the signals work:
--   In IF/ELSIF/ELSE chains with variable assignments, pgcov injects a NOTIFY
--   *inside* each branch (after the assignment).  That NOTIFY only fires when
--   the BRANCH ABOVE IT is taken.  So "out_of_stock" coverage signal fires only
--   when stock = 0; "low_stock" signal fires only when 0 < stock <= 10; etc.

-- DO $$
-- DECLARE
--     v_id    INT;
--     v_price NUMERIC;
-- BEGIN
--     -- get_stock_status: triggers the 'out_of_stock' branch signal
--     v_id := add_product('Empty Shelf', 1.00, 0);
--     ASSERT get_stock_status(v_id) = 'out_of_stock',
--         format('Expected out_of_stock, got %s', get_stock_status(v_id));
--
--     -- get_stock_status: triggers the 'low_stock' branch signal
--     v_id := add_product('Rare Find', 1.00, 5);
--     ASSERT get_stock_status(v_id) = 'low_stock',
--         format('Expected low_stock, got %s', get_stock_status(v_id));
--
--     -- apply_pricing_policy: premium tier (15 % off)
--     --   triggers the 'standard' ELSIF signal (fires inside the premium branch)
--     v_id := add_product('Priced Item', 100.00, 20);
--     v_price := apply_pricing_policy(v_id, 'premium');
--     ASSERT v_price = 85.00,
--         format('Expected 85.00, got %s', v_price);
--
--     -- apply_pricing_policy: standard tier (5 % off)
--     --   triggers the 'none' ELSE signal (fires inside the standard branch)
--     v_price := apply_pricing_policy(v_id, 'standard');
--     ASSERT v_price = 95.00,
--         format('Expected 95.00, got %s', v_price);
--
--     RAISE NOTICE 'All branch tests passed -- 100%% coverage achieved!';
-- END $$;
