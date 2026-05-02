---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: cmd/sqleton/cmds/db.go
      Note: removed old driver import
    - Path: go.mod
      Note: dependency cleanup after go mod tidy
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Investigation Diary: DuckDB Duplicate Symbol Linker Failure

## 2026-05-02 — Initial Report and Reproduction

**Reported:** Sqleton fails to compile. User suspects a prior fix in `../clay` (avoiding linking a different DuckDB version) did not fully resolve the issue.

**Reproduction command:**
```bash
make build
```

**Observed output (truncated):**
```
go generate ./...
go build -tags sqlite_fts5 -ldflags "-X main.version=v0.4.3-2a40fc2-" ./...
# github.com/go-go-golems/sqleton/cmd/sqleton
.../link: running gcc failed: exit status 1
/usr/bin/ld: .../libduckdb_static.a(ub_duckdb_main_capi.cpp.o): in function `duckdb::CAPIAggregateDestructor(...)':
multiple definition of `duckdb::CAPIAggregateDestructor(...)'; .../libduckdb.a(ub_duckdb_main_capi.cpp.o): first defined here
# (hundreds of similar lines)
```

**Initial hypothesis:** Two DuckDB libraries are being linked simultaneously.

---

## 2026-05-02 — Dependency Graph Analysis

Ran `go mod graph | grep -i duckdb` to see which modules depend on which DuckDB packages.

**Key findings:**
- `github.com/go-go-golems/sqleton` directly requires `github.com/marcboeker/go-duckdb v1.8.5`
- `github.com/go-go-golems/clay` (local workspace replacement) requires `github.com/duckdb/duckdb-go/v2 v2.10501.0`
- Both the old and new drivers embed their own static C libraries (`libduckdb.a` vs `libduckdb_static.a`)

**Command:**
```bash
go mod why github.com/duckdb/duckdb-go/v2
# github.com/go-go-golems/clay/pkg/sql
# github.com/duckdb/duckdb-go/v2

go mod why github.com/marcboeker/go-duckdb
# github.com/go-go-golems/sqleton/cmd/sqleton/cmds
# github.com/marcboeker/go-duckdb
```

**Confirmed:** The old driver is pulled in by sqleton's own code; the new driver is pulled in by clay.

---

## 2026-05-02 — Source Code Inspection

**File:** `cmd/sqleton/cmds/db.go`

Found the offending blank import:
```go
_ "github.com/marcboeker/go-duckdb" // DuckDB driver for database/sql
```

**File:** `../clay/pkg/sql/config.go`

Found that clay already imports the new driver:
```go
_ "github.com/duckdb/duckdb-go/v2"
```

Since `db.go` imports `sql2 "github.com/go-go-golems/clay/pkg/sql"`, the new driver is already transitively included. The old import is redundant and harmful.

---

## 2026-05-02 — Workspace Confirmation

**Command:**
```bash
go env GOWORK
# /home/manuel/code/wesen/corporate-headquarters/go.work
```

This explains why sqleton picks up the migrated clay code even though sqleton's `go.mod` still pins `clay v0.4.3` (which used the old driver). The workspace overrides version resolution.

---

## 2026-05-02 — Fix Applied

**Change:** Removed the old blank import from `cmd/sqleton/cmds/db.go`.

**Diff:**
```diff
- 	_ "github.com/go-sql-driver/mysql"  // MySQL driver for database/sql
- 	_ "github.com/marcboeker/go-duckdb" // DuckDB driver for database/sql
+ 	_ "github.com/go-sql-driver/mysql" // MySQL driver for database/sql
```

**Module cleanup:**
```bash
go mod tidy
```

This moved `marcboeker/go-duckdb` from `require` (direct) to an indirect dependency, but more importantly it is no longer compiled into the binary because nothing in the build graph imports it anymore.

---

## 2026-05-02 — Verification

**Command:**
```bash
make build
```

**Result:**
```
go generate ./...
go build -tags sqlite_fts5 -ldflags "-X main.version=v0.4.3-2a40fc2-dirty" ./...
```

Build completed successfully. No linker errors.

---

## What Worked

- Using `go mod why` to trace dependency paths quickly identified the two sources of DuckDB.
- Using `rg` to find the exact import line in source code made the fix obvious.
- Checking `go env GOWORK` confirmed the workspace hypothesis.
- Removing a single blank import and running `go mod tidy` was sufficient.

## What Did Not Work

- `go mod tidy` alone would not have fixed this because the problem was a source-level import, not a stale `go.mod` entry.
- Upgrading `clay` in isolation without checking sibling workspace modules left sqleton in a broken state.

## What Was Tricky

- The error output is enormous (hundreds of "multiple definition" lines from `ld`), which can be intimidating. The trick is to scroll to the top and read the file paths in the linker command — they reveal which two libraries are colliding.
- The `go.mod` of sqleton still lists `clay v0.4.3`, which uses the old driver. Without knowing about the workspace override, one might incorrectly conclude that clay is not the source of the new driver.

## Code Review Instructions

When reviewing this fix:

1. Verify `cmd/sqleton/cmds/db.go` no longer imports `marcboeker/go-duckdb`.
2. Verify `go.mod` no longer lists `marcboeker/go-duckdb` in the direct `require` block.
3. Run `make build` locally and confirm a clean build.
4. Optional: Run `go list -m all | grep duckdb` and confirm only `duckdb/duckdb-go/v2` and its bindings remain in the active module list.

## Unresolved / Follow-up

- **CI gap:** We do not have automated workspace-wide builds. A change in `clay` can break `sqleton`, `escuse-me`, or other dependents without anyone noticing until they run `make build` locally.
- **Future driver upgrades:** When `duckdb-go/v2` releases a new version, we should upgrade it in `clay` first, then verify all workspace members.
- **Documentation:** This ticket now serves as the canonical reference for diagnosing C static library collisions in our Go workspace.
