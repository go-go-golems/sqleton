---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ../../../../../../../clay/pkg/sql/config.go
      Note: clay file importing the new driver
    - Path: ../../../../../../../go.work
      Note: workspace file causing local clay resolution
    - Path: cmd/sqleton/cmds/db.go
      Note: source file that contained the old driver import
    - Path: go.mod
      Note: module manifest showing dependency change
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Sqleton DuckDB Linker Conflict: Deep Dive and Fix Guide for New Interns

## 1. Executive Summary

**Sqleton** is our Go CLI tool for running SQL queries against multiple database backends (MySQL, PostgreSQL, SQLite, DuckDB). Recently, running `make build` in the sqleton repository started failing with hundreds of linker errors of the form `multiple definition of 'duckdb_*'`. This document explains, from first principles, why this happened and how we fixed it. If you are a new intern, read this to understand our Go monorepo workspace, how Cgo-based database drivers work, and how to diagnose dependency diamond problems.

**The fix in one sentence:** Remove the direct import of the old DuckDB driver (`github.com/marcboeker/go-duckdb`) from sqleton's source code, because our shared `clay` library had already migrated to the new official driver (`github.com/duckdb/duckdb-go/v2`), and linking two different static DuckDB libraries simultaneously causes symbol collisions.

---

## 2. What is Sqleton?

Sqleton (`github.com/go-go-golems/sqleton`) is a command-line SQL runner built on top of our internal frameworks:

- **Glazed** (`go-go-golems/glazed`) — provides structured command parsing, parameter layers, row processing, and output formatting (JSON, YAML, tables).
- **Clay** (`go-go-golems/clay`) — provides shared utilities, especially database connection management (`clay/pkg/sql`).
- **Parka** (`go-go-golems/parka`) — optional web/REST frontend layer.

Sqleton itself is relatively thin: it defines Cobra CLI commands, reads SQL query definitions from YAML files, and executes them through Go's standard `database/sql` interface. The actual heavy lifting (parsing connection strings, managing connection pools, formatting results) is delegated to Glazed and Clay.

### 2.1 Key files in sqleton

| File | Purpose |
|------|---------|
| `cmd/sqleton/main.go` | Entry point. Bootstraps Cobra, loads commands, starts the CLI. |
| `cmd/sqleton/cmds/db.go` | Database management subcommands (`db test`, `db ls`, `db print-env`, etc.). This is where database drivers are imported as blank imports for side-effect registration. |
| `pkg/cmds/*.go` | Core command definitions, SQL parameter parsing, query execution. |
| `go.mod` | Go module manifest. Declares direct and transitive dependencies. |
| `Makefile` | Build automation. Runs `go generate` and `go build -tags sqlite_fts5`. |

---

## 3. The Go Workspace: Why `../clay` Matters

Our organization maintains multiple Go modules in a single Git repository (a **monorepo**). The directory structure looks like this:

```
corporate-headquarters/
├── clay/          # github.com/go-go-golems/clay
├── glazed/        # github.com/go-go-golems/glazed
├── parka/         # github.com/go-go-golems/parka
├── sqleton/       # github.com/go-go-golems/sqleton
└── go.work        # Go workspace file
```

### 3.1 What is a Go workspace?

A Go workspace (file: `go.work`) tells the Go toolchain: "When you are inside this directory tree, resolve module imports from these local directories instead of downloading them from the internet."

Our workspace file contains:

```go
// corporate-headquarters/go.work
go 1.26.2

use (
    ./clay
    ./glazed
    ./parka
    ./sqleton
    // ... many others
)
```

**Consequence:** Even though sqleton's `go.mod` declares `require github.com/go-go-golems/clay v0.4.3`, the Go compiler actually uses the code in `../clay/` because the workspace says so. This is critical for day-to-day development: we can change Clay, Glazed, and Sqleton in lockstep without publishing new versions.

**The trap:** Because the workspace silently redirects module resolution, a change in `../clay` can break `../sqleton` even if sqleton's `go.mod` has not changed. That is exactly what happened here.

---

## 4. What is DuckDB and Why Do We Use It?

**DuckDB** is an in-process analytical database (similar to SQLite but optimized for OLAP workloads). It is written in C++. We use it because:

- It can query CSV, Parquet, and JSON files directly with SQL.
- It is embedded — no separate server process.
- It has excellent performance for analytical queries.

### 4.1 How Go talks to DuckDB

Go cannot directly call C++ code. The bridge is built via **Cgo** and the **DuckDB C API**:

```
+----------------------------------+
|  Your Go code (sqleton)          |
|  import "database/sql"           |
+----------------------------------+
           |
           v
+----------------------------------+
|  Go database/sql driver          |
|  (github.com/duckdb/duckdb-go/v2)|
+----------------------------------+
           |
           v (Cgo calls)
+----------------------------------+
|  DuckDB C API (libduckdb)        |
|  (C header + static library)     |
+----------------------------------+
           |
           v
+----------------------------------+
|  DuckDB engine (C++)             |
|  (inside libduckdb_static.a)     |
+----------------------------------+
```

The Go driver package contains C header files and a pre-compiled **static library** (`.a` file) for each platform:
- `lib/linux-amd64/libduckdb_static.a` on Linux
- `lib/darwin-arm64/libduckdb_static.a` on macOS

When Go builds a binary that imports the driver, the Go linker (`ld`) embeds this static library into the final executable. This is why DuckDB-enabled Go binaries are large (~100+ MB) — they contain the entire database engine.

---

## 5. The Compile Error: Exactly What Happened

Running `make build` produced a wall of text ending with:

```
/usr/bin/ld: /home/manuel/go/pkg/mod/github.com/duckdb/duckdb-go-bindings/lib/linux-amd64@v0.10501.0/libduckdb_static.a(ub_duckdb_main_capi.cpp.o): in function `duckdb::CAPIAggregateDestructor(...)':
multiple definition of `duckdb::CAPIAggregateDestructor(...)'; 
/home/manuel/go/pkg/mod/github.com/marcboeker/go-duckdb@v1.8.5/deps/linux_amd64/libduckdb.a(ub_duckdb_main_capi.cpp.o): first defined here

/usr/bin/ld: ... multiple definition of `duckdb_appender_create'; ...
/usr/bin/ld: ... multiple definition of `duckdb_append_bool'; ...
# (hundreds more lines like this)
```

The key observation: the linker is complaining about **two different files**:

1. `marcboeker/go-duckdb@v1.8.5/deps/linux_amd64/libduckdb.a` — the **old** driver
2. `duckdb/duckdb-go-bindings/lib/linux-amd64@v0.10501.0/libduckdb_static.a` — the **new** driver

Both are DuckDB static libraries, both define the same C symbols (like `duckdb_appender_create`), and the linker refuses to link both into one binary.

---

## 6. Root Cause Analysis: The Dependency Diamond

### 6.1 The two DuckDB drivers

| Package | Author | Status | DuckDB Version |
|---------|--------|--------|----------------|
| `github.com/marcboeker/go-duckdb` | Community (Marc Boeker) | Legacy / Old | v1.8.x |
| `github.com/duckdb/duckdb-go/v2` | Official DuckDB Labs | Current | v2.105.x |

The DuckDB organization recently released an official Go driver. Our `clay` library migrated to the official driver. Sqleton did not.

### 6.2 The dependency graph

Because of the Go workspace, the actual resolved graph looks like this:

```
                    sqleton binary
                         |
         +---------------+---------------+
         |                               |
         v                               v
   sqleton/cmd/db.go              clay/pkg/sql
   (local source)                 (from ../clay/)
         |                               |
   _ "marcboeker/go-duckdb"      _ "duckdb/duckdb-go/v2"
         |                               |
         v                               v
   libduckdb.a                  libduckdb_static.a
   (old static lib)             (new static lib)
   same C symbols!              same C symbols!
         \                               /
          \______________+______________/
                         |
                    LINKER SAYS NO
                    "multiple definition"
```

### 6.3 Why did this work before?

Before the clay migration, both sqleton and clay imported `marcboeker/go-duckdb`. There was only one static library, so the linker was happy. When clay was upgraded to the official driver, the workspace caused sqleton to pick up the new clay code, but sqleton still imported the old driver. Two libraries entered the link step; the linker rejected them.

### 6.4 Why is this a "diamond dependency"?

A diamond dependency occurs when two of your dependencies each depend on a different version of the same underlying library. In Go, this is usually fine because Go's module graph selects a single version (Minimal Version Selection). **But** when the "library" is a C static library embedded inside a Go package, Go's version selection cannot merge two different `.a` files. The linker sees both and fails.

---

## 7. How We Diagnosed It

Here is the exact debugging process, step by step, so you can reproduce it on similar issues.

### 7.1 Step 1: Read the error carefully

Look at the file paths in the linker output. We saw two distinct module cache paths:
- `marcboeker/go-duckdb@v1.8.5/...`
- `duckdb/duckdb-go-bindings/lib/linux-amd64@v0.10501.0/...`

This immediately tells us two different packages are pulling in two different DuckDB libraries.

### 7.2 Step 2: Find who imports each package

```bash
# Which package needs the NEW driver?
go mod why github.com/duckdb/duckdb-go/v2
# Output:
# github.com/go-go-golems/clay/pkg/sql
# github.com/duckdb/duckdb-go/v2

# Which package needs the OLD driver?
go mod why github.com/marcboeker/go-duckdb
# Output:
# github.com/go-go-golems/sqleton/cmd/sqleton/cmds
# github.com/marcboeker/go-duckdb
```

### 7.3 Step 3: Find the import in source code

```bash
rg -n "marcboeker/go-duckdb" cmd/ pkg/ --type go
# Output:
# cmd/sqleton/cmds/db.go:20:    _ "github.com/marcboeker/go-duckdb"
```

### 7.4 Step 4: Check the workspace

```bash
go env GOWORK
# Output: /home/manuel/code/wesen/corporate-headquarters/go.work

cat /home/manuel/code/wesen/corporate-headquarters/go.work | grep clay
# Output: ./clay
```

This confirms that sqleton is using the local `../clay` (which has the new driver) rather than the tagged v0.4.3 from the module cache (which still had the old driver).

### 7.5 Step 5: Verify the driver registration in clay

```bash
cat ../clay/pkg/sql/config.go | grep "duckdb-go/v2"
# Output:
# _ "github.com/duckdb/duckdb-go/v2"
```

Clay already imports the new driver as a blank import, which means `database/sql` will already have the `"duckdb"` driver registered whenever `clay/pkg/sql` is imported. Sqleton does not need to import any DuckDB driver itself.

---

## 8. The Fix

### 8.1 Code change

File: `cmd/sqleton/cmds/db.go`

**Before:**
```go
import (
    // ... other imports ...
    _ "github.com/go-sql-driver/mysql"  // MySQL driver for database/sql
    _ "github.com/marcboeker/go-duckdb" // DuckDB driver for database/sql
)
```

**After:**
```go
import (
    // ... other imports ...
    _ "github.com/go-sql-driver/mysql" // MySQL driver for database/sql
    // NOTE: DuckDB driver is imported transitively via clay/pkg/sql
)
```

We simply removed the blank import of `github.com/marcboeker/go-duckdb`.

### 8.2 Module cleanup

After removing the import, run:

```bash
go mod tidy
```

This updates `go.mod` and `go.sum` to remove the now-unused direct dependency on `marcboeker/go-duckdb`. It may still appear as an **indirect** dependency if some other transitive dependency uses it, but it will no longer be compiled into the sqleton binary.

### 8.3 Verify the build

```bash
make build
```

Expected output:
```
go generate ./...
go build -tags sqlite_fts5 -ldflags "-X main.version=v0.4.3-2a40fc2-dirty" ./...
```

No linker errors. Success.

---

## 9. Why the Fix Works

When we removed the old import, the Go compiler no longer included the old static library (`libduckdb.a` from `marcboeker/go-duckdb`) in the link step. Only the new static library (`libduckdb_static.a` from `duckdb-go-bindings`) remains. All DuckDB C symbols now have exactly one definition, and the linker succeeds.

Because `clay/pkg/sql` (which sqleton already imports) includes `_ "github.com/duckdb/duckdb-go/v2"`, the `database/sql` driver named `"duckdb"` is still registered at program startup. Sqleton can still open DuckDB connections via `sql.Open("duckdb", dsn)` without any code changes elsewhere.

---

## 10. Prevention: How to Avoid This in the Future

### 10.1 When upgrading shared libraries, audit all workspace members

If you upgrade a Cgo-based dependency in `clay`, `glazed`, or `parka`, check every module in the workspace (`go.work`) that depends on it. Look for:

- Direct imports of the old package.
- `go.mod` lines that pin the old package as a direct dependency.

Use this command to scan:

```bash
# In the monorepo root
for dir in clay glazed parka sqleton escuse-me; do
  echo "=== $dir ==="
  (cd $dir && rg -l "marcboeker/go-duckdb" --type go 2>/dev/null)
done
```

### 10.2 Prefer transitive driver registration

If a shared library (`clay/pkg/sql`) already registers a database driver, application code (`sqleton`) should not import the driver again. This is a standard Go pattern: the lowest-level shared package registers the driver; higher-level packages just use `database/sql`.

### 10.3 After any `go.mod` change in a workspace member, build siblings

Our CI should ideally build all workspace members when any shared dependency changes. Locally, you can run:

```bash
# From monorepo root
for dir in sqleton escuse-me parka; do
  (cd $dir && make build)
done
```

### 10.4 Understand when `go mod tidy` is not enough

`go mod tidy` fixes module metadata, but it cannot fix **source-level import mismatches**. If two of your imports each embed a different C static library with the same symbols, `go mod tidy` will happily keep both in `go.mod`. You must fix the imports in `.go` source files.

---

## 11. Concepts Glossary for New Interns

| Term | Definition |
|------|------------|
| **Cgo** | Go's FFI (Foreign Function Interface) that lets Go code call C code. Used by database drivers that wrap C/C++ libraries. |
| **Static library** | A `.a` file (Unix) or `.lib` file (Windows) containing compiled object code. Linked directly into the final executable. |
| **Linker** | The tool (usually `ld` on Linux) that combines compiled object files and libraries into a single executable. It errors if the same symbol is defined twice. |
| **Blank import** | `import _ "package"` — imports a package only for its side effects (typically `init()` functions, such as driver registration). |
| **Go workspace** | A `go.work` file that groups multiple Go modules so they resolve each other locally instead of from the module proxy. |
| **MVS** | Minimal Version Selection — Go's algorithm for picking dependency versions. It does not help with C library collisions. |
| **Driver registration** | In Go's `database/sql`, drivers call `sql.Register("name", driverInstance)` in their `init()` so `sql.Open("name", ...)` works. |

---

## 12. References

- **Sqleton source file changed:** `cmd/sqleton/cmds/db.go`
- **Sqleton module manifest:** `go.mod`
- **Monorepo workspace:** `/home/manuel/code/wesen/corporate-headquarters/go.work`
- **Clay SQL package (new driver):** `../clay/pkg/sql/config.go`
- **New official DuckDB Go driver:** `github.com/duckdb/duckdb-go/v2` (resolved in module cache at `~/go/pkg/mod/github.com/duckdb/duckdb-go/v2@v2.10501.0`)
- **Old community DuckDB Go driver:** `github.com/marcboeker/go-duckdb` (resolved in module cache at `~/go/pkg/mod/github.com/marcboeker/go-duckdb@v1.8.5`)
- **DuckDB C API docs:** https://duckdb.org/docs/api/c/overview
