---
doc_type: design-doc
title: "DuckDB Integration: Architecture Analysis and Implementation Guide"
ticket: SQLETON-03-DUCKDB-SUPPORT
status: active
intent: long-term
topics: [backend, duckdb, database]
created: 2026-04-05
authors: [pi]
---

# DuckDB Integration: Architecture Analysis and Implementation Guide

**Ticket**: SQLETON-03-DUCKDB-SUPPORT
**Audience**: New intern joining the project. This document is self-contained and assumes no prior knowledge of sqleton, glazed, or the go-go-golems ecosystem.

---

## 1. Executive Summary

Sqleton is a Go-based CLI tool that runs SQL queries against relational databases and formats the results as beautiful tables, JSON, CSV, or YAML. It currently supports MySQL, PostgreSQL (via pgx), and SQLite. The goal of this ticket is to add **DuckDB** as a fourth supported database backend.

DuckDB is an in-process analytical database engine (think "the SQLite of OLAP"). It is particularly well-suited for querying large CSV, Parquet, and JSON files directly, making it a natural fit for sqleton's use case of quick ad-hoc data exploration. The integration requires changes across two repositories in the go workspace: the `clay` library (which owns the SQL connection layer) and the `sqleton` CLI itself.

**Why this matters**: Adding DuckDB support turns sqleton into a powerful tool for local analytical workloads. A user can point sqleton at a directory of Parquet files and run SQL queries immediately, with zero setup, leveraging DuckDB's columnar engine for fast aggregation and filtering.

---

## 2. Problem Statement and Scope

### 2.1 What We Want

- Users can pass `--db-type duckdb` (or `--db-type duck`) on the command line to connect to a DuckDB database file.
- Users can pass `--dsn "mydata.duckdb"` to open a DuckDB file.
- Users can use `--dsn ""` (empty DSN) to open an in-memory DuckDB instance for quick throwaway queries.
- DuckDB works with all existing sqleton commands: `query`, `run`, `select`, `db test`, and SQL command files loaded from repositories.
- DuckDB-specific template functions (e.g., for reading Parquet/CSV) are available in SQL command files.
- The serve (HTTP) mode and MCP tool mode work with DuckDB connections.

### 2.2 What Is Out of Scope (for Phase 1)

- No DuckDB-specific UI or special result handling.
- No automatic detection of file format (the user specifies `.duckdb` files explicitly).
- No integration with DuckDB's extensions system (beyond what the Go driver supports by default).
- No dbt profile integration for DuckDB (dbt-duckdb is a separate adapter).

---

## 3. Current-State Architecture

### 3.1 Workspace Layout

The project uses a Go workspace (`go.work`) with four modules:

```
sqleton-duckdb-glm/
├── go.work                    # Go workspace file
├── .ttmp.yaml                 # docmgr config
├── sqleton/                   # The sqleton CLI (main module)
│   ├── cmd/sqleton/           # CLI entrypoint and Cobra commands
│   ├── pkg/cmds/              # SQL command loading, compilation, execution
│   ├── pkg/flags/             # SQL helper flag definitions
│   ├── pkg/codegen/           # Code generation from SQL files
│   └── go.mod                 # Dependencies (mysql, pgx, sqlite3 drivers)
├── clay/                      # Shared library (SQL connection layer lives here)
│   └── pkg/sql/               # Database config, connection, query execution
├── glazed/                    # Output formatting framework
└── go-minitrace/              # Unrelated project (transcript analysis)
```

**Key insight**: The database connection logic lives in `clay/pkg/sql/`, not in sqleton itself. Sqleton imports clay and delegates all connection management to it.

### 3.2 How a SQL Query Flows Through the System

Here is the complete execution path when a user runs:

```bash
sqleton query --db-type mysql --host localhost --user root --database mydb \
  "SELECT id, name FROM users LIMIT 5"
```

```
User types command
       │
       ▼
┌─────────────────────────────────────┐
│  main.go: rootCmd.Execute()        │  Cobra CLI framework dispatches
│  Routes to QueryCommand            │  to the "query" subcommand
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  cmds/query.go: QueryCommand       │  Implements cmds.GlazeCommand
│  .RunIntoGlazeProcessor()          │  interface
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  sql.DBConnectionFactory           │  A function type:
│  (clay/pkg/sql/settings.go:42)     │  func(ctx, *values.Values) (*sqlx.DB, error)
│                                     │
│  Default impl:                      │
│  OpenDatabaseFromDefaultSqlConnectionLayer()
│  (clay/pkg/sql/settings.go:44)     │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  DatabaseConfig.Connect()          │  (clay/pkg/sql/config.go:115)
│                                     │  1. Resolves connection string
│  1. GetSource() → *Source           │  2. Normalizes driver name
│  2. sqlx.Open(dbType, connStr)     │  3. Opens via database/sql + driver
│  3. db.PingContext(ctx)            │  4. Pings to verify connection
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  sql.RunNamedQueryIntoGlaze()      │  (clay/pkg/sql/query.go:35)
│                                     │  1. db.PrepareNamedContext()
│  1. Prepares statement              │  2. stmt.QueryxContext()
│  2. Executes query                  │  3. rows.MapScan() for each row
│  3. Scans results into rows         │  4. Converts []byte → string
│  4. Feeds rows to GlazeProcessor   │  5. gp.AddRow(ctx, row)
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  Glazed Output Pipeline            │  Middlewares process rows:
│  (glazed framework)                 │  - Field filtering
│                                     │  - Row filtering
│  Final output: table/JSON/CSV/YAML │  - Column reordering
└─────────────────────────────────────┘
```

### 3.3 The Database Connection Layer (Clay)

The connection layer is the heart of the system for this ticket. It lives in `clay/pkg/sql/` and consists of these key files:

| File | Purpose |
|------|---------|
| `clay/pkg/sql/settings.go` | `SqlConnectionParameterLayer`, `DbtParameterLayer`, `DBConnectionFactory` type, `OpenDatabaseFromDefaultSqlConnectionLayer()` |
| `clay/pkg/sql/config.go` | `DatabaseConfig` struct, `Connect()` method, `GetSource()`, driver normalization |
| `clay/pkg/sql/sources.go` | `Source` struct, `ToConnectionString()`, `ParseDbtProfiles()` |
| `clay/pkg/sql/query.go` | `RunQueryIntoGlaze()`, `RunNamedQueryIntoGlaze()`, `RenderQuery()` |
| `clay/pkg/sql/template.go` | SQL template functions (`sqlStringIn`, `sqlDate`, `sqlColumn`, etc.) |
| `clay/pkg/sql/flags/sql-connection.yaml` | YAML definition of connection flags (`host`, `database`, `db-type`, `dsn`, etc.) |
| `clay/pkg/sql/flags/dbt.yaml` | YAML definition of dbt flags |

#### 3.3.1 The `DatabaseConfig` Struct

```go
// clay/pkg/sql/config.go (lines ~18-34)
type DatabaseConfig struct {
    Host            string `glazed:"host"`
    Database        string `glazed:"database"`
    User            string `glazed:"user"`
    Password        string `glazed:"password"`
    Port            int    `glazed:"port"`
    Schema          string `glazed:"schema"`
    Type            string `glazed:"db-type"`
    DSN             string `glazed:"dsn"`
    Driver          string `glazed:"driver"`
    SSLDisable      bool   `glazed:"ssl-disable"`
    DbtProfilesPath string `glazed:"dbt-profiles-path"`
    DbtProfile      string `glazed:"dbt-profile"`
    UseDbtProfiles  bool   `glazed:"use-dbt-profiles"`
}
```

This struct is populated from the parsed command-line flags (via the glazed parameter system). The `glazed` struct tags map flag names to struct fields.

#### 3.3.2 Driver Normalization in `GetSource()`

```go
// clay/pkg/sql/config.go (lines ~93-109) — current normalization logic
switch strings.ToLower(source.Type) {
case "sqlite":
    source.Type = "sqlite3"
case "postgres", "postgresql", "pg":
    source.Type = "pgx"
case "mariadb":
    source.Type = "mysql"
}
```

**This is where we need to add `duckdb` and `duck` as aliases.** The `source.Type` value is passed directly to `sqlx.Open()` as the driver name, so it must match the Go `database/sql` driver's registered name.

#### 3.3.3 Connection String Building in `Source.ToConnectionString()`

```go
// clay/pkg/sql/sources.go (lines ~23-36)
func (s *Source) ToConnectionString() string {
    switch s.Type {
    case "pgx":
        return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", ...)
    case "mysql":
        return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", ...)
    case "sqlite", "sqlite3":
        return s.Database    // just the file path
    default:
        return ""            // ← currently returns empty for unknown types!
    }
}
```

**For DuckDB, the connection string is the file path** (similar to SQLite), so we need to add a `case "duckdb"` that returns the database file path.

#### 3.3.4 The `Connect()` Method

```go
// clay/pkg/sql/config.go (lines ~115-170)
func (c *DatabaseConfig) Connect(ctx context.Context) (*sqlx.DB, error) {
    // 1. If DSN is set, infer driver from DSN scheme
    // 2. Normalize driver aliases
    // 3. Call sqlx.Open(dbType, connectionString)
    // 4. Ping with 5-second timeout
    // 5. Return *sqlx.DB
}
```

The `Connect()` method already supports DSN-based connections with driver inference from the scheme. We need to add DuckDB scheme detection here too.

### 3.4 SQL Command Files

Sqleton can load SQL commands from `.sql` files that have a special YAML preamble:

```sql
/* sqleton
name: ls-posts-type [types...]
short: "Show all WP posts, limited, by type"
flags:
  - name: limit
    type: int
    default: 10
    help: Limit the number of posts
arguments:
  - name: types
    type: stringList
    default: ["post", "page"]
*/
SELECT wp.ID, wp.post_title FROM wp_posts wp
WHERE post_type IN ({{ .types | sqlStringIn }})
LIMIT {{ .limit }}
```

**How loading works**:

1. `SqlCommandLoader.LoadCommands()` (in `sqleton/pkg/cmds/loaders.go`) detects file type from extension.
2. `ParseSQLFileSpecFromReader()` (in `sqleton/pkg/cmds/spec.go`) splits the `/* sqleton ... */` preamble from the SQL body.
3. `SqlCommandCompiler.Compile()` creates a `SqlCommand` struct with the parsed metadata.
4. The `SqlCommand` (in `sqleton/pkg/cmds/sql.go`) holds the query template and executes it at runtime.

**Template functions** available in queries are defined in `clay/pkg/sql/template.go` and include:
- `sqlStringIn` — format a string list for `IN (...)`
- `sqlDate` / `sqlDateTime` — date formatting
- `sqliteDate` / `sqliteDateTime` — SQLite-specific date formatting
- `sqlLike` — LIKE pattern matching
- `sqlColumn` / `sqlSingle` / `sqlMap` / `sqlSlice` — execute sub-queries within templates

### 3.5 The Parameter Layer System

Sqleton uses a layered parameter system (provided by glazed) where values come from multiple sources in priority order:

```
1. Command-line flags    (highest priority)
2. Environment variables (SQLETON_HOST, SQLETON_DATABASE, etc.)
3. Config file           (--config-file)
4. Profile settings      (~/.config/sqleton/profiles.yaml)
5. Defaults              (lowest priority)
```

Each "section" (e.g., `sql-connection`, `dbt`, `sql-helpers`, `glazed`) defines its own set of flags via YAML files. The sections are composed together on each command.

The middleware chain is assembled in `sqleton/pkg/cmds/cobra.go`:
- `sources.FromCobra()` — reads flags from the Cobra command
- `sources.FromEnv()` — reads SQLETON_* environment variables
- `sources.FromFiles()` — reads config file
- `sources.GatherFlagsFromProfiles()` — reads profile settings
- `sources.FromDefaults()` — fills in defaults

### 3.6 Database Drivers Currently Registered

Drivers are imported via blank imports (Go's `database/sql` side-effect registration pattern):

```go
// sqleton/cmd/sqleton/cmds/db.go:28
_ "github.com/go-sql-driver/mysql"    // MySQL

// clay/pkg/sql/config.go:8-10
_ "github.com/go-sql-driver/mysql"    // MySQL
_ "github.com/jackc/pgx/v5/stdlib"   // PostgreSQL
_ "github.com/mattn/go-sqlite3"       // SQLite
```

**Note**: `go-sqlite3` requires CGo, which means the binary must be compiled with a C compiler. The DuckDB Go driver (`github.com/marcboeker/go-duckdb`) also requires CGo for the same reason.

### 3.7 Key Data Flow Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                     USER (CLI / HTTP)                         │
└──────────────┬───────────────────────────────────────────────┘
               │
    ┌──────────┼──────────────────────────────────────┐
    │          ▼                                      │
    │  ┌─────────────────┐   ┌─────────────────┐     │
    │  │  Cobra Command  │   │  SQL .sql file   │     │
    │  │  (query/run/    │   │  (loaded from    │     │
    │  │   select)       │   │   repository)    │     │
    │  └────────┬────────┘   └────────┬────────┘     │
    │           │                     │               │  sqleton/
    │           ▼                     ▼               │
    │  ┌─────────────────────────────────────┐        │
    │  │     SqlCommand / QueryCommand       │        │
    │  │     (RunIntoGlazeProcessor)         │        │
    │  └────────────────┬────────────────────┘        │
    │                   │                              │
    └───────────────────┼─────────────────────────────┘
                        │
                        ▼
         ┌─────────────────────────────┐
         │  DBConnectionFactory        │  clay/pkg/sql/
         │  ┌─────────────────────┐    │
         │  │ DatabaseConfig      │    │
         │  │ .Connect()          │    │
         │  │   ↓                 │    │
         │  │ sqlx.Open(driver,   │    │
         │  │   connectionString) │    │
         │  └─────────────────────┘    │
         └──────────────┬──────────────┘
                        │
          ┌─────────────┼──────────────────┐
          ▼             ▼                  ▼
    ┌──────────┐  ┌──────────┐  ┌──────────────┐
    │  MySQL   │  │  PostgreSQL│  │   SQLite     │
    │  driver  │  │  pgx      │  │  sqlite3     │
    └──────────┘  └──────────┘  └──────────────┘
```

---

## 4. Gap Analysis

### 4.1 What's Missing for DuckDB Support

| Gap | Location | Severity |
|-----|----------|----------|
| No DuckDB driver import | `clay/pkg/sql/config.go`, `sqleton/cmd/sqleton/cmds/db.go` | Critical |
| No `duckdb`/`duck` alias in driver normalization | `clay/pkg/sql/config.go:GetSource()` | Critical |
| No `duckdb` case in `ToConnectionString()` | `clay/pkg/sql/sources.go` | Critical |
| No DSN scheme detection for `duckdb://` | `clay/pkg/sql/config.go:Connect()` | Medium |
| No `duckdbDate`/`duckdbDateTime` template functions | `clay/pkg/sql/template.go` | Low (DuckDB uses standard SQL dates) |
| No DuckDB example queries in repository | `sqleton/cmd/sqleton/queries/` | Low |
| `go.mod` does not include DuckDB dependency | Both `clay/go.mod` and `sqleton/go.mod` | Critical |
| `sql-connection.yaml` help text doesn't mention DuckDB | `clay/pkg/sql/flags/sql-connection.yaml` | Low |

### 4.2 Compatibility Considerations

- **CGo**: The most popular Go DuckDB driver (`github.com/marcboeker/go-duckdb`) uses CGo, just like `go-sqlite3`. This is consistent with the existing build requirements.
- **Connection model**: DuckDB uses a file path as its connection string (like SQLite), but also supports `:memory:` for in-memory databases. This maps well to the existing `Database` field.
- **No host/port/user/password**: DuckDB is an in-process engine. It doesn't use network connections or authentication. The `Host`, `Port`, `User`, `Password` fields are irrelevant but should not cause errors.
- **Transaction semantics**: DuckDB supports standard SQL transactions. The existing `RunQueryIntoGlaze` / `RunNamedQueryIntoGlaze` functions should work without modification.
- **Data types**: DuckDB returns Go types that are compatible with `sqlx.MapScan()`. The existing `processQueryResults()` in `clay/pkg/sql/query.go` handles `[]byte → string` conversion, which should work for DuckDB too.

---

## 5. Proposed Architecture and APIs

### 5.1 Approach: Minimal Extension of the Existing Pattern

DuckDB fits naturally into the existing driver abstraction. The approach is to treat DuckDB as "just another driver" — the same pattern used for MySQL, PostgreSQL, and SQLite. No architectural changes are needed.

### 5.2 Driver Selection

**Which Go DuckDB driver?** The recommended driver is:

```
github.com/marcboeker/go-duckdb/v2
```

This is the most mature Go `database/sql`-compatible driver for DuckDB. It supports:
- File-based and in-memory databases
- Prepared statements
- `database/sql` interface (required for `sqlx`)
- Parquet, CSV, JSON file reading via SQL
- CGo-based (links against DuckDB's C library)

**Alternative driver**: `github.com/tanimutomo/go-duckdb` — less maintained, not recommended.

### 5.3 Connection String Format

DuckDB's connection string is simply the file path:

```go
// File-based database
connStr := "/path/to/mydata.duckdb"

// In-memory database
connStr := ""

// Read-only mode
connStr := "/path/to/mydata.duckdb?access_mode=read_only"
```

This maps to sqleton's `--database` flag (file path) or `--dsn` flag.

---

## 6. Pseudocode and Key Flows

### 6.1 Driver Registration (blank import)

In `clay/pkg/sql/config.go`, add the import:

```go
import (
    _ "github.com/marcboeker/go-duckdb/v2"  // DuckDB driver
    // ... existing imports ...
)
```

In `sqleton/cmd/sqleton/cmds/db.go`, also add (for commands that directly open DBs):

```go
import (
    _ "github.com/marcboeker/go-duckdb/v2"  // DuckDB driver
    // ... existing imports ...
)
```

### 6.2 Driver Normalization in `GetSource()`

Add DuckDB aliases to the switch statement in `clay/pkg/sql/config.go`:

```go
// Current (around line 93):
switch strings.ToLower(source.Type) {
case "sqlite":
    source.Type = "sqlite3"
case "postgres", "postgresql", "pg":
    source.Type = "pgx"
case "mariadb":
    source.Type = "mysql"
// NEW:
case "duckdb", "duck":
    source.Type = "duckdb"
}
```

### 6.3 Connection String in `ToConnectionString()`

Add a case for DuckDB in `clay/pkg/sql/sources.go`:

```go
func (s *Source) ToConnectionString() string {
    switch s.Type {
    case "pgx":
        return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", ...)
    case "mysql":
        return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", ...)
    case "sqlite", "sqlite3":
        return s.Database
    // NEW:
    case "duckdb":
        return s.Database  // file path or "" for in-memory
    default:
        return ""
    }
}
```

### 6.4 DSN Scheme Detection in `Connect()`

Add DuckDB scheme detection to the driver inference block in `clay/pkg/sql/config.go`:

```go
// Current DSN scheme detection (around line 130):
case strings.HasPrefix(lower, "postgres://"), strings.HasPrefix(lower, "postgresql://"):
    c.Driver = "pgx"
case strings.HasPrefix(lower, "mysql://"), strings.HasPrefix(lower, "mariadb://"):
    c.Driver = "mysql"
case strings.HasPrefix(lower, "sqlite://"), strings.HasPrefix(lower, "sqlite3://"):
    c.Driver = "sqlite3"
// NEW:
case strings.HasPrefix(lower, "duckdb://"):
    c.Driver = "duckdb"
```

Also add to the alias normalization:

```go
// Current driver alias normalization (around line 140):
case "postgres", "postgresql", "pg":
    c.Driver = "pgx"
case "sqlite":
    c.Driver = "sqlite3"
case "mariadb":
    c.Driver = "mysql"
// NEW:
case "duckdb", "duck":
    c.Driver = "duckdb"
```

### 6.5 Updated `sql-connection.yaml` Help Text

In `clay/pkg/sql/flags/sql-connection.yaml`, update the `db-type` help:

```yaml
  - name: db-type
    type: string
    help: "Database type (mysql, pgx, sqlite3, duckdb)"
    default: mysql
    shortFlag: t
```

### 6.6 DuckDB-Specific Example Queries

Create `sqleton/cmd/sqleton/queries/duckdb/` with example queries:

```sql
-- queries/duckdb/parquet-tables.sql
/* sqleton
name: parquet-tables
short: "List tables from Parquet files in a directory"
flags:
  - name: path
    type: string
    help: "Path to directory or glob pattern"
    required: true
*/
SELECT table_name, estimated_size
FROM duckdb_tables()
WHERE table_name LIKE '%parquet%'
ORDER BY table_name
```

```sql
-- queries/duckdb/read-csv.sql
/* sqleton
name: read-csv
short: "Query a CSV file using DuckDB"
flags:
  - name: file_path
    type: string
    help: "Path to CSV file"
    required: true
  - name: limit
    type: int
    default: 50
    help: "Max rows to return"
*/
SELECT * FROM read_csv_auto('{{ .file_path }}')
LIMIT {{ .limit }}
```

```sql
-- queries/duckdb/read-parquet.sql
/* sqleton
name: read-parquet
short: "Query a Parquet file using DuckDB"
flags:
  - name: file_path
    type: string
    help: "Path to Parquet file"
    required: true
  - name: limit
    type: int
    default: 50
    help: "Max rows to return"
*/
SELECT * FROM read_parquet('{{ .file_path }}')
LIMIT {{ .limit }}
```

```sql
-- queries/duckdb/list-files.sql
/* sqleton
name: list-files
short: "List files that DuckDB can read from a directory"
flags:
  - name: directory
    type: string
    help: "Directory path to list"
    required: true
  - name: extension
    type: string
    default: "*.parquet"
    help: "File extension glob pattern"
*/
SELECT file
FROM glob('{{ .directory }}/{{ .extension }}')
ORDER BY file
```

### 6.7 Complete Execution Flow with DuckDB

Here is how the flow works end-to-end after the changes:

```bash
# User runs:
sqleton query --db-type duckdb --database ./analytics.duckdb \
  "SELECT * FROM read_parquet('data/*.parquet') LIMIT 10"
```

```
1. Cobra parses flags:
   - db-type = "duckdb"
   - database = "./analytics.duckdb"
   - query = "SELECT * FROM ..."

2. DBConnectionFactory called:
   - ParsedValues contain sql-connection section with db-type="duckdb"

3. DatabaseConfig populated:
   - Type = "duckdb"
   - Database = "./analytics.duckdb"

4. GetSource() normalizes:
   - "duckdb" → "duckdb" (passes through)

5. ToConnectionString():
   - case "duckdb" → returns "./analytics.duckdb"

6. sqlx.Open("duckdb", "./analytics.duckdb"):
   - go-duckdb driver opens the file
   - Returns *sqlx.DB

7. PingContext():
   - DuckDB responds to ping

8. RunNamedQueryIntoGlaze():
   - Prepares statement
   - Executes query
   - Scans results via MapScan
   - Feeds rows to GlazeProcessor

9. Output: formatted table/JSON/CSV
```

---

## 7. Implementation Phases

### Phase 1: Core Driver Integration (clay)

**Goal**: Make clay recognize and connect to DuckDB.

**Files to modify**:

1. **`clay/pkg/sql/config.go`**
   - Add `import _ "github.com/marcboeker/go-duckdb/v2"` (line ~9)
   - Add `"duckdb", "duck"` to driver normalization in `GetSource()` (~line 106)
   - Add `"duckdb://"` DSN scheme detection in `Connect()` (~line 133)
   - Add `"duckdb", "duck"` to driver alias normalization in `Connect()` (~line 146)

2. **`clay/pkg/sql/sources.go`**
   - Add `case "duckdb":` to `ToConnectionString()` (~line 30)

3. **`clay/pkg/sql/flags/sql-connection.yaml`**
   - Update `db-type` help text to include `duckdb`

4. **`clay/go.mod`**
   - Add `github.com/marcboeker/go-duckdb/v2` dependency
   - Run `go mod tidy`

**Verification**:
```bash
cd clay && go build ./pkg/sql/...
```

### Phase 2: Driver Registration in Sqleton

**Goal**: Ensure the DuckDB driver is imported in the sqleton binary.

**Files to modify**:

1. **`sqleton/cmd/sqleton/cmds/db.go`**
   - Add `import _ "github.com/marcboeker/go-duckdb/v2"` (line ~28)

2. **`sqleton/go.mod`**
   - Add the indirect dependency (will be pulled in via clay)
   - Run `go mod tidy`

**Verification**:
```bash
cd sqleton && go build ./cmd/sqleton/...
```

### Phase 3: Example Queries and Documentation

**Goal**: Provide DuckDB-specific SQL command files and update docs.

**Files to create**:

1. **`sqleton/cmd/sqleton/queries/duckdb/parquet-tables.sql`**
2. **`sqleton/cmd/sqleton/queries/duckdb/read-csv.sql`**
3. **`sqleton/cmd/sqleton/queries/duckdb/read-parquet.sql`**
4. **`sqleton/cmd/sqleton/queries/duckdb/list-files.sql`**
5. **`sqleton/cmd/sqleton/doc/topics/duckdb.md`** (help topic)

**Files to modify**:

1. **`sqleton/README.md`** — Add DuckDB examples

### Phase 4: Integration Testing

**Goal**: Verify DuckDB works end-to-end with all sqleton commands.

**Tests to write**:

1. **Unit test in clay**: `clay/pkg/sql/config_test.go`
   - Test driver normalization (`duckdb` → `duckdb`, `duck` → `duckdb`)
   - Test `ToConnectionString()` returns file path for DuckDB
   - Test DSN scheme detection for `duckdb://`

2. **Unit test in sqleton**: `sqleton/cmd/sqleton/cmds/query_test.go`
   - Test `sqleton query --db-type duckdb --database :memory: "SELECT 1"`
   - Test `sqleton run` with a .sql file against in-memory DuckDB
   - Test `sqleton select --table information_schema.tables --db-type duckdb --database :memory:`

3. **Integration test**:
   - Create a temporary DuckDB file with test data
   - Run queries against it
   - Verify output in multiple formats

**Verification**:
```bash
# Manual smoke test
cd sqleton && go run ./cmd/sqleton query \
  --db-type duckdb --database "" \
  "SELECT 42 AS answer"

# Test with a Parquet file (if available)
go run ./cmd/sqleton query \
  --db-type duckdb --database "" \
  "SELECT * FROM read_parquet('test/testdata/sample.parquet') LIMIT 5"
```

---

## 8. Testing and Validation Strategy

### 8.1 Unit Tests

| Test | File | What It Verifies |
|------|------|------------------|
| `TestGetSource_DuckDB` | `clay/pkg/sql/config_test.go` | Driver normalization for `duckdb` and `duck` |
| `TestToConnectionString_DuckDB` | `clay/pkg/sql/sources_test.go` | Returns file path for DuckDB type |
| `TestConnect_DuckDB_Memory` | `clay/pkg/sql/config_test.go` | Can open in-memory DuckDB |
| `TestConnect_DuckDB_File` | `clay/pkg/sql/config_test.go` | Can open file-based DuckDB |
| `TestDSNScheme_DuckDB` | `clay/pkg/sql/config_test.go` | DSN `duckdb://path` resolves correctly |

### 8.2 Integration Tests

```bash
# Test 1: Basic query
sqleton query --db-type duckdb --database "" "SELECT 1 AS test"

# Test 2: JSON output
sqleton query --db-type duckdb --database "" --output json "SELECT 1 AS test"

# Test 3: Run from file
echo "SELECT * FROM generate_series(1, 10)" > /tmp/test.sql
sqleton run --db-type duckdb --database "" /tmp/test.sql

# Test 4: Select command
sqleton select --table duckdb_settings --db-type duckdb --database ""

# Test 5: DB test command
sqleton db test --db-type duckdb --database ""

# Test 6: SQL command file with template
cat > /tmp/duckdb-test.sql << 'EOF'
/* sqleton
name: duckdb-range
short: "Generate a range of integers"
flags:
  - name: start
    type: int
    default: 1
  - name: stop
    type: int
    default: 10
*/
SELECT * FROM generate_series({{ .start }}, {{ .stop }})
EOF
sqleton run-command /tmp/duckdb-test.sql -- --db-type duckdb --database "" --start 5 --stop 15
```

### 8.3 DuckDB-Specific Test Data

For testing file format reading, create a small Parquet file:

```bash
# Using DuckDB CLI (if installed):
duckdb -c "COPY (SELECT i AS id, 'item_' || i AS name, random() * 100 AS price FROM generate_series(1, 100) t(i)) TO 'testdata/sample.parquet' (FORMAT PARQUET)"
```

---

## 9. Risks, Alternatives, and Open Questions

### 9.1 Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| DuckDB CGo dependency causes build issues on some platforms | Medium | High | Document CGo requirement; provide Docker build fallback |
| DuckDB driver API incompatibility with `sqlx` | Low | High | The `go-duckdb/v2` driver implements `database/sql` interface; `sqlx` wraps this transparently |
| DuckDB returns types not handled by `processQueryResults()` | Low | Medium | DuckDB returns standard Go types; test with `MapScan()` |
| Large result sets exhaust memory | Medium | Medium | DuckDB is analytical; add `--limit` recommendation in docs |
| Breaking existing MySQL/Postgres/SQLite connections | Low | High | Changes are additive (new `case` branches); existing paths untouched |

### 9.2 Alternatives Considered

1. **Pure-Go DuckDB driver** (no CGo): Does not exist as of 2026-04. DuckDB's core is C++ and all Go drivers use CGo.

2. **Separate `sqleton-duckdb` binary**: Could avoid pulling CGo dependency into the main binary. Rejected because it fragments the user experience and duplicates code.

3. **Build tag to make DuckDB optional**: Could use `//go:build duckdb` to make the driver import conditional. This is a good future optimization but not needed for Phase 1.

4. **Using DuckDB's native Go API instead of `database/sql`**: Would bypass `sqlx` and require significant refactoring. The `database/sql` driver approach is far simpler.

### 9.3 Open Questions

1. **Should `--db-type duckdb` use a default port?** No — DuckDB is in-process and doesn't use ports. The existing default port (3306) should not interfere because it's only used for MySQL/Postgres connection string building.

2. **Should empty `--database` default to in-memory DuckDB?** Yes. When `db-type` is `duckdb` and `database` is empty, the connection string should be `""` (in-memory).

3. **Should we add DuckDB-specific template functions?** Not in Phase 1. DuckDB uses standard SQL date functions (`DATE '2024-01-01'`), so the existing `sqlDate` function works. Phase 2 could add `duckdbDate` for explicit DuckDB date formatting if needed.

4. **How to handle DuckDB extensions?** DuckDB can load extensions (`INSTALL httpfs; LOAD httpfs;`) via SQL commands. These can be run as normal queries through sqleton. No special support needed.

---

## 10. References

### 10.1 Key Source Files

| File | Purpose |
|------|---------|
| `clay/pkg/sql/config.go` | Database config, `Connect()` method, driver normalization |
| `clay/pkg/sql/sources.go` | `Source.ToConnectionString()`, dbt profile parsing |
| `clay/pkg/sql/settings.go` | `DBConnectionFactory` type, parameter layers |
| `clay/pkg/sql/query.go` | Query execution into GlazeProcessor |
| `clay/pkg/sql/template.go` | SQL template functions |
| `clay/pkg/sql/flags/sql-connection.yaml` | Connection flag definitions |
| `sqleton/cmd/sqleton/main.go` | CLI entrypoint, command initialization |
| `sqleton/cmd/sqleton/cmds/db.go` | `db` subcommand, driver imports |
| `sqleton/cmd/sqleton/cmds/query.go` | `query` command implementation |
| `sqleton/cmd/sqleton/cmds/run.go` | `run` command implementation |
| `sqleton/cmd/sqleton/cmds/select.go` | `select` command implementation |
| `sqleton/cmd/sqleton/cmds/serve.go` | HTTP server mode |
| `sqleton/cmd/sqleton/cmds/mcp/mcp.go` | MCP tool integration |
| `sqleton/pkg/cmds/sql.go` | `SqlCommand` struct, query rendering and execution |
| `sqleton/pkg/cmds/spec.go` | SQL file spec parsing (preamble extraction) |
| `sqleton/pkg/cmds/loaders.go` | Command loading from `.sql` files |
| `sqleton/pkg/cmds/cobra.go` | Middleware chain assembly |
| `sqleton/pkg/flags/settings.go` | SQL helpers parameter layer |

### 10.2 External References

- DuckDB: https://duckdb.org/
- go-duckdb driver: https://github.com/marcboeker/go-duckdb
- sqlx: https://jmoiron.github.io/sqlx/
- glazed framework: https://github.com/go-go-golems/glazed
- Go database/sql: https://pkg.go.dev/database/sql

### 10.3 Existing Driver Pattern Examples

For reference, here is how SQLite support works end-to-end (the closest analogy to DuckDB):

1. **Driver import**: `clay/pkg/sql/config.go:10` — `_ "github.com/mattn/go-sqlite3"`
2. **Normalization**: `config.go:99` — `case "sqlite": source.Type = "sqlite3"`
3. **Connection string**: `sources.go:33-34` — `case "sqlite": fallthrough; case "sqlite3": return s.Database`
4. **CLI usage**: `sqleton query --db-type sqlite --database ./my.db "SELECT * FROM table"`
5. **Example queries**: `sqleton/cmd/sqleton/queries/sqlite/` — `tables.sql`, `hishtory/ls.sql`

DuckDB should follow this exact same pattern.

---

## Appendix A: Glossary

| Term | Definition |
|------|-----------|
| **sqleton** | A Go CLI tool that runs SQL queries and formats results |
| **clay** | A shared Go library providing SQL connection management |
| **glazed** | A Go framework for CLI output formatting (tables, JSON, CSV, etc.) |
| **parka** | A Go framework for serving glazed commands over HTTP |
| **DuckDB** | An in-process analytical (OLAP) database engine, similar to SQLite but optimized for analytics |
| **sqlx** | A Go library that extends `database/sql` with struct scanning and named parameter support |
| **Cobra** | A Go CLI framework (spf13/cobra) used for command-line parsing |
| **CGo** | Go's foreign function interface for calling C code; required by some database drivers |
| **DSN** | Data Source Name — a connection string that identifies a database |
| **OLAP** | Online Analytical Processing — a category of database optimized for complex queries on large datasets |
| **Parquet** | A columnar storage file format commonly used in data analytics |
| **`database/sql`** | Go's standard library interface for SQL databases; drivers register themselves via `init()` |
| **Blank import** | `import _ "pkg"` — imports a package solely for its side effects (driver registration) |
| **Section/Parameter Layer** | A named group of CLI flags managed by the glazed framework |
| **GlazeCommand** | The interface that all sqleton commands implement: `RunIntoGlazeProcessor(ctx, values, processor) error` |
| **GlazeProcessor** | The middleware pipeline that transforms rows into formatted output |
| **Middleware** | A function that modifies or enriches parsed parameter values before command execution |
| **Preamble** | The YAML metadata block at the top of a `.sql` file (between `/* sqleton` and `*/`) |
| **Repository** | A directory of `.sql` command files that sqleton loads at startup |

## Appendix B: sqleton CLI Command Reference

| Command | Description | Relevant File |
|---------|-------------|---------------|
| `sqleton query <SQL>` | Execute a SQL query string | `cmds/query.go` |
| `sqleton run <file...>` | Execute SQL from files or stdin | `cmds/run.go` |
| `sqleton run-command <file>` | Run a SQL command file with its own flags | `main.go` (`runCommandCmd`) |
| `sqleton select` | Build and execute a SELECT query | `cmds/select.go` |
| `sqleton db test` | Test database connection | `cmds/db.go` |
| `sqleton db ls` | List databases from dbt profiles | `cmds/db.go` |
| `sqleton db print-settings` | Print connection settings | `cmds/db.go` |
| `sqleton db print-env` | Print connection as env vars | `cmds/db.go` |
| `sqleton serve` | Start HTTP server | `cmds/serve.go` |
| `sqleton mcp tools list` | List MCP tools | `cmds/mcp/mcp.go` |
| `sqleton mcp tools run` | Run an MCP tool | `cmds/mcp/mcp.go` |
| `sqleton codegen <files>` | Generate Go code from SQL files | `cmds/codegen.go` |

## Appendix C: Connection String Formats

| Database | Driver Name | Connection String Format | Example |
|----------|-------------|-------------------------|---------|
| MySQL | `mysql` | `user:pass@tcp(host:port)/dbname` | `root:mypass@tcp(localhost:3306)/mydb` |
| PostgreSQL | `pgx` | `host=X port=N user=X password=X dbname=X sslmode=X` | `host=localhost port=5432 user=postgres dbname=mydb sslmode=require` |
| SQLite | `sqlite3` | File path | `./data.db` |
| DuckDB (proposed) | `duckdb` | File path or empty (in-memory) | `./analytics.duckdb` or `` (in-memory) |
