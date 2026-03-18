# Demo Example: Product Inventory System

This example shows **pgcov** catching uncovered branches in PL/pgSQL functions — a
situation that happens all the time in real projects.

## What's Here

| File | Description |
| ------ | ------------- |
| `01_schema.sql` | `products` table + seed data |
| `02_functions.sql` | Three PL/pgSQL functions with input-validation branches |
| `demo_test.sql` | Tests — initially happy-path only, expandable to full coverage |

## The Schema

A simple product inventory with three management functions:

```
add_product(name, price, stock, category) → INT
    Branches: empty name | negative price | negative stock | happy path

get_stock_status(product_id) → TEXT
    Branches: not found ('unknown') | stock = 0 ('out_of_stock')
            | stock ≤ 10 ('low_stock') | else ('in_stock')

apply_discount(product_id, discount_pct) → NUMERIC
    Branches: discount out of range | product not found | happy path
```

## Demo Walkthrough

### Step 1 — Look at the schema

Browse `01_schema.sql` and `02_functions.sql`.  
Each function has several `IF` branches. Which ones are actually exercised?

### Step 2 — Run pgcov (partial coverage)

```bash
# From the repository root
pgcov run -c "postgresql://user@host/postgres?sslmode=disable" ./examples/demo/
pgcov report --format html -o report-partial.html
```

Open `report-partial.html`. The validation branches are highlighted — they are
**never reached** by the basic happy-path tests.

### Step 3 — Uncomment the edge-case tests

In `demo_test.sql`, remove the `--` prefix from every line of the second `DO` block
(the one that starts with `-- DO $$`).

### Step 4 — Run pgcov again (full coverage)

```bash
pgcov run -c "postgresql://user@host/postgres?sslmode=disable" ./examples/demo/
pgcov report --format html -o report-full.html
```

Open `report-full.html`. Every branch in every function is now covered.

## Quick Start

```bash
# Build the tool (once)
go build ./cmd/pgcov

# --- Partial run ---
./pgcov run -c "postgresql://pasha@127.0.0.1/postgres?sslmode=disable" ./examples/demo/
./pgcov report --format html -o report-partial.html

# Edit demo_test.sql: uncomment the second DO block

# --- Full run ---
./pgcov run -c "postgresql://pasha@127.0.0.1/postgres?sslmode=disable" ./examples/demo/
./pgcov report --format html -o report-full.html
```
