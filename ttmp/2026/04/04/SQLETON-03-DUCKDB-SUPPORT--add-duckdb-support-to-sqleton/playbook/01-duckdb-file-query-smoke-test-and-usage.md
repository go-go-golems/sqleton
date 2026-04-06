---
Title: DuckDB file-query smoke test and usage
Ticket: SQLETON-03-DUCKDB-SUPPORT
Status: active
Topics:
    - backend
    - duckdb
    - database
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: sqleton/README.md
      Note: Public-facing DuckDB usage examples and connection semantics
    - Path: sqleton/cmd/sqleton/doc/topics/02-database-sources.md
      Note: DuckDB connection model and file-query guidance
    - Path: sqleton/cmd/sqleton/doc/topics/07-duckdb-file-queries.md
      Note: Dedicated DuckDB file-query help topic
ExternalSources: []
Summary: |
    Reproducible playbook for validating and explaining sqleton's DuckDB-backed workflow for querying JSON, CSV, and Parquet files directly from SQL.
LastUpdated: 2026-04-05T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# DuckDB file-query smoke test and usage

## Purpose

Provide a repeatable operator workflow for proving that sqleton can use DuckDB
to query raw JSON, CSV, and Parquet files directly, without importing them into
MySQL or PostgreSQL first.

## Environment Assumptions

- Repository root: `/home/manuel/workspaces/2026-04-04/sqleton-duckdb-glm`
- `clay/` and `sqleton/` both build successfully.
- The Go toolchain can build CGo-based drivers.
- DuckDB support is present in the current sqleton binary/module state.

## Commands

### 1. Create a temporary fixture directory

```bash
set -euo pipefail
TMPDIR=$(mktemp -d /tmp/sqleton-duckdb-smoke.XXXXXX)
mkdir -p "$TMPDIR/json" "$TMPDIR/csv"

echo "TMPDIR=$TMPDIR"
```

### 2. Create JSON and CSV fixtures

```bash
cat > "$TMPDIR/json/events-1.json" <<'EOF'
[
  {"user_id": 1, "event": "login", "country": "US", "amount": 10},
  {"user_id": 2, "event": "purchase", "country": "DE", "amount": 25},
  {"user_id": 1, "event": "purchase", "country": "US", "amount": 15}
]
EOF

cat > "$TMPDIR/json/events-2.json" <<'EOF'
[
  {"user_id": 3, "event": "login", "country": "FR", "amount": 0},
  {"user_id": 2, "event": "purchase", "country": "DE", "amount": 30},
  {"user_id": 1, "event": "refund", "country": "US", "amount": -5}
]
EOF

cat > "$TMPDIR/csv/sales-1.csv" <<'EOF'
region,product,amount,qty
US,widget,100,2
DE,widget,80,1
US,gadget,60,3
EOF

cat > "$TMPDIR/csv/sales-2.csv" <<'EOF'
region,product,amount,qty
FR,widget,50,1
DE,gadget,120,4
US,widget,40,1
EOF

find "$TMPDIR" -maxdepth 2 -type f | sort
```

### 3. Query JSON files through DuckDB

```bash
cd /home/manuel/workspaces/2026-04-04/sqleton-duckdb-glm/sqleton

go run ./cmd/sqleton query --db-type duckdb --database '' --output json \
  "SELECT user_id, SUM(amount) AS total_amount, COUNT(*) AS event_count
   FROM read_json_auto('$TMPDIR/json/*.json', format='array')
   GROUP BY user_id
   ORDER BY user_id"
```

Expected result shape:
- user `1` total `20`, count `3`
- user `2` total `55`, count `2`
- user `3` total `0`, count `1`

### 4. Query CSV files through DuckDB

```bash
go run ./cmd/sqleton query --db-type duckdb --database '' --output json \
  "SELECT region, SUM(amount) AS revenue, SUM(qty) AS units
   FROM read_csv_auto('$TMPDIR/csv/*.csv')
   GROUP BY region
   ORDER BY region"
```

Expected result shape:
- `DE` revenue `200`, units `5`
- `FR` revenue `50`, units `1`
- `US` revenue `200`, units `6`

### 5. Generate and query a Parquet file

```bash
go run ./cmd/sqleton query --db-type duckdb --database '' \
  "COPY (SELECT * FROM read_csv_auto('$TMPDIR/csv/*.csv'))
   TO '$TMPDIR/out.parquet' (FORMAT PARQUET)"

go run ./cmd/sqleton query --db-type duckdb --database '' --output json \
  "SELECT product, SUM(amount) AS revenue, SUM(qty) AS units
   FROM read_parquet('$TMPDIR/out.parquet')
   GROUP BY product
   ORDER BY product"
```

Expected result shape:
- `gadget` revenue `180`, units `7`
- `widget` revenue `270`, units `5`

## Exit Criteria

The playbook is successful when all of the following are true:

1. `go run ./cmd/sqleton ...` succeeds for all three data formats.
2. JSON file globs are read with `read_json_auto(..., format='array')`.
3. CSV file globs are read with `read_csv_auto(...)`.
4. The generated Parquet file is read successfully with `read_parquet(...)`.
5. The query outputs match the expected aggregation totals shown above.

## Notes

- The DuckDB connection itself is configured via `--db-type duckdb` and
  `--database ''`.
- The file paths belong inside the SQL, not in the `--database` flag.
- `--database ''` means an in-memory DuckDB instance.
- A persistent DuckDB file would look like `--database ./analytics.duckdb`.
- If sqleton emits startup warnings about unrelated embedded aliases, those are
  currently noisy but do not block the DuckDB workflow.
