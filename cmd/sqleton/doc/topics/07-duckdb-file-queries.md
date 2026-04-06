---
Title: DuckDB file queries
Slug: duckdb-file-queries
Short: Query JSON, CSV, and Parquet files directly through sqleton using DuckDB.
Topics:
- duckdb
- files
- analytics
Commands:
- query
- run
- db
Flags:
- db-type
- database
- output
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

## Overview

DuckDB support lets sqleton act as a lightweight analytics CLI for local files.
Instead of loading JSON, CSV, or Parquet data into a server database first, you
connect sqleton to DuckDB and use DuckDB SQL functions to read files directly.

The common pattern is:

1. Start a DuckDB connection with `--db-type duckdb`.
2. Use `--database ''` for an in-memory engine, or `--database ./file.duckdb`
   for a persistent local database file.
3. Reference the files inside the SQL statement with DuckDB functions such as
   `read_json_auto`, `read_csv_auto`, and `read_parquet`.

## Quick examples

### Query JSON arrays

```bash
sqleton query --db-type duckdb --database '' --output json \
  "SELECT user_id, SUM(amount) AS total_amount, COUNT(*) AS event_count
   FROM read_json_auto('./events/*.json', format='array')
   GROUP BY user_id
   ORDER BY user_id"
```

### Query CSV files

```bash
sqleton query --db-type duckdb --database '' --output json \
  "SELECT region, SUM(amount) AS revenue, SUM(qty) AS units
   FROM read_csv_auto('./reports/*.csv')
   GROUP BY region
   ORDER BY region"
```

### Query Parquet files

```bash
sqleton query --db-type duckdb --database '' --output json \
  "SELECT product, SUM(amount) AS revenue
   FROM read_parquet('./warehouse/*.parquet')
   GROUP BY product
   ORDER BY product DESC"
```

## Connection model

DuckDB in sqleton does **not** treat a JSON/CSV/Parquet glob as the sqleton
`--database` value. Instead:

- `--database ''` means: create an in-memory DuckDB connection.
- `--database ./analytics.duckdb` means: open a persistent DuckDB database file.
- `read_json_auto('./events/*.json')` means: read external files from SQL.

So this is the intended pattern:

```bash
sqleton query --db-type duckdb --database '' \
  "SELECT * FROM read_csv_auto('./data/*.csv') LIMIT 10"
```

and **not** this:

```bash
# Not the intended usage model
sqleton query --db-type duckdb --database './data/*.csv' "SELECT ..."
```

## When to use in-memory vs persistent DuckDB

### In-memory DuckDB

Use this when you want fast ad hoc inspection of raw files:

```bash
sqleton query --db-type duckdb --database '' \
  "SELECT COUNT(*) FROM read_json_auto('./events/*.json', format='array')"
```

### Persistent DuckDB database file

Use this when you want to cache results, create tables, or reuse derived data:

```bash
sqleton query --db-type duckdb --database ./analytics.duckdb \
  "CREATE TABLE IF NOT EXISTS daily_sales AS
   SELECT * FROM read_parquet('./warehouse/sales/*.parquet')"
```

## Useful DuckDB file functions

### `read_json_auto`

Best for JSON data when you want DuckDB to infer the schema.

```sql
SELECT * FROM read_json_auto('./events/*.json', format='array')
```

If each file contains a top-level array, `format='array'` is often the right
choice.

### `read_csv_auto`

Best for CSV input with header rows and inferred types.

```sql
SELECT * FROM read_csv_auto('./exports/*.csv')
```

### `read_parquet`

Best for Parquet files and Parquet globs.

```sql
SELECT * FROM read_parquet('./warehouse/*.parquet')
```

## Smoke-test pattern

A small end-to-end validation loop looks like this:

```bash
# 1. JSON
sqleton query --db-type duckdb --database '' --output json \
  "SELECT user_id, COUNT(*)
   FROM read_json_auto('./events/*.json', format='array')
   GROUP BY user_id"

# 2. CSV
sqleton query --db-type duckdb --database '' --output json \
  "SELECT region, SUM(amount)
   FROM read_csv_auto('./reports/*.csv')
   GROUP BY region"

# 3. Parquet
sqleton query --db-type duckdb --database '' --output json \
  "SELECT product, SUM(revenue)
   FROM read_parquet('./warehouse/*.parquet')
   GROUP BY product"
```

## Recommended sqleton usage notes

- Use `--output json` when you want the result to feed another tool.
- Use `--output csv` when you want a quick export after aggregating raw files.
- Prefer `query` for ad hoc SQL and `run` for repeatable SQL files checked into a repo.
- Keep file paths in the SQL layer; keep connection settings in sqleton flags.
