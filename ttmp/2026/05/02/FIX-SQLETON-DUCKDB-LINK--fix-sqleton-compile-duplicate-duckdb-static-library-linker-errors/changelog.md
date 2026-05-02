# Changelog

## 2026-05-02

- Initial workspace created


## 2026-05-02

Diagnosed and fixed duplicate DuckDB static library linker errors by removing old driver import from cmd/sqleton/cmds/db.go and running go mod tidy

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/sqleton/cmd/sqleton/cmds/db.go — removed _ github.com/marcboeker/go-duckdb import
- /home/manuel/code/wesen/corporate-headquarters/sqleton/go.mod — go mod tidy removed direct dependency on marcboeker/go-duckdb

