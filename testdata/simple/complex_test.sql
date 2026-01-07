-- Test complex function
SELECT calculate_discount(100, 'VIP') = 80 AS test_vip_discount;
SELECT calculate_discount(100, 'REGULAR') = 90 AS test_regular_discount;
SELECT calculate_discount(100, 'GUEST') = 95 AS test_guest_discount;
