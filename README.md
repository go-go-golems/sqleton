# ☠️ sqleton ☠️ - Powerful SQL CLI Tool with Rich Output

[![golangci-lint](https://github.com/wesen/sqleton/actions/workflows/lint.yml/badge.svg)](https://github.com/wesen/sqleton/actions/workflows/lint.yml)
[![golang-pipeline](https://github.com/wesen/sqleton/actions/workflows/push.yml/badge.svg)](https://github.com/wesen/sqleton/actions/workflows/push.yml)

![sqleton logo](doc/logo.png)

**sqleton** is a command-line tool that makes SQL execution fast, flexible, and beautifully formatted. Execute queries, manage database connections, and export data in multiple formats with professional output quality suitable for both development and business use.

Built on the powerful [glazed](https://github.com/go-go-golems/glazed) framework, sqleton combines the speed of command-line tools with the rich formatting capabilities of modern data processing applications.

## Quick Start

```bash
# Execute a simple query
sqleton query --db-type mysql --host localhost --user root --database mydb \
  "SELECT id, name, email FROM users LIMIT 5"

# Get JSON output for API integration
sqleton query --db-type postgres --host localhost --database analytics \
  --output json "SELECT category, SUM(revenue) FROM sales GROUP BY category"

# Generate CSV report for Excel
sqleton query --db-type sqlite --database ./data.db \
  --output csv "SELECT date, orders_count, revenue FROM daily_stats" > report.csv

# Query a directory of JSON files with DuckDB
sqleton query --db-type duckdb --database '' --output json \
  "SELECT user_id, COUNT(*) AS events
   FROM read_json_auto('./events/*.json', format='array')
   GROUP BY user_id
   ORDER BY events DESC"
```

## Core Features

### Multiple Database Support
Connect to MySQL, PostgreSQL, SQLite, and DuckDB databases with a consistent interface and authentication/connection model appropriate for each backend.

DuckDB support is especially useful for local analytics workflows because sqleton can use DuckDB to query external JSON, CSV, and Parquet files directly from SQL without importing them into another server database first.

### Rich Output Formats
Professional formatting for every use case:

**Table** (default) - Perfect for terminal inspection:
```
+----+----------+---------------------+
| id | username | email               |
+----+----------+---------------------+
| 1  | johndoe  | john@example.com    |
| 2  | janesmit | jane@example.com    |
+----+----------+---------------------+
```

**JSON** - Ideal for APIs and data integration:
```json
[
  {"id": 1, "username": "johndoe", "email": "john@example.com"},
  {"id": 2, "username": "janesmit", "email": "jane@example.com"}
]
```

**CSV** - Ready for spreadsheets and BI tools:
```csv
id,username,email
1,johndoe,john@example.com
2,janesmit,jane@example.com
```

### YAML Command Definitions
Create reusable, parameterized SQL commands with rich templating:

```yaml
name: user-report [status...]
short: Generate user analytics report with filtering
flags:
  - name: limit
    type: int
    default: 50
    help: Maximum number of users to include
  - name: min_orders
    type: int
    help: Filter users with minimum order count
arguments:
  - name: status
    type: stringList
    default: ["active"]
    help: User status filter
query: |
  SELECT u.id, u.username, u.email, COUNT(o.id) as order_count
  FROM users u 
  LEFT JOIN orders o ON u.id = o.user_id
  WHERE u.status IN ({{ .status | sqlStringIn }})
  {{ if .min_orders -}}
  AND order_count >= {{ .min_orders }}
  {{- end }}
  GROUP BY u.id
  ORDER BY order_count DESC
  LIMIT {{ .limit }}
```

### Flexible Connection Management
Multiple ways to specify database connections:

- **Command-line flags**: Direct specification for one-off queries
- **Environment variables**: Secure credential management
- **Configuration files**: Shareable connection profiles
- **DBT profiles**: Integration with existing dbt workflows

### Built-in Database Commands
Common database operations without writing SQL:

```bash
# List tables in database
sqleton db ls-tables

# Test database connection
sqleton db test

# Describe table structure
sqleton select --table users --describe

# Quick data inspection with filtering
sqleton select --table products --where "price > 100" --columns name,price
```

## Installation

### Installing the Framework
To use sqleton as a library in your Go project:
```bash
go get github.com/go-go-golems/sqleton
```

### Installing the `sqleton` CLI Tool

**Using Homebrew:**
```bash
brew tap go-go-golems/go-go-go
brew install go-go-golems/go-go-go/sqleton
```

**Using apt-get:**
```bash
echo "deb [trusted=yes] https://apt.fury.io/go-go-golems/ /" >> /etc/apt/sources.list.d/fury.list
apt-get update
apt-get install sqleton
```

**Using yum:**
```bash
echo "
[fury]
name=Gemfury Private Repo
baseurl=https://yum.fury.io/go-go-golems/
enabled=1
gpgcheck=0
" >> /etc/yum.repos.d/fury.repo
yum install sqleton
```

**Using go install:**
```bash
go install github.com/go-go-golems/sqleton/cmd/sqleton@latest
```

**Download binaries from [GitHub Releases](https://github.com/go-go-golems/sqleton/releases)**

**Or run from source:**
```bash
go run ./cmd/sqleton
```

## Live Demo

Want to see sqleton in action? Try our MySQL demo with realistic ecommerce data:

### Setup Demo Environment
```bash
# Start MySQL with sample data
docker run --name sqleton-demo \
  -e MYSQL_ROOT_PASSWORD=demo123 \
  -e MYSQL_DATABASE=ecommerce \
  -p 3306:3306 \
  -d mysql:8.0

# Wait for startup, then load sample data
sleep 30
curl -fsSL https://raw.githubusercontent.com/go-go-golems/sqleton/main/examples/demo-data.sql | \
  docker exec -i sqleton-demo mysql -u root -pdemo123 ecommerce
```

### Try Demo Queries

**Basic customer analysis:**
```bash
sqleton query --db-type mysql --host localhost --user root --password demo123 \
  --database ecommerce --port 3306 \
  "SELECT username, email, status FROM users WHERE status = 'active' LIMIT 5"
```

**Business analytics with JSON output:**
```bash
sqleton query --db-type mysql --host localhost --user root --password demo123 \
  --database ecommerce --port 3306 --output json \
  "SELECT category, COUNT(*) as products, AVG(price) as avg_price 
   FROM products GROUP BY category ORDER BY avg_price DESC"
```

**Daily revenue report as CSV:**
```bash
sqleton query --db-type mysql --host localhost --user root --password demo123 \
  --database ecommerce --port 3306 --output csv \
  "SELECT DATE(order_date) as day, COUNT(*) as orders, SUM(total_amount) as revenue 
   FROM orders GROUP BY DATE(order_date) ORDER BY day"
```

**Customer spending analysis:**
```bash
sqleton query --db-type mysql --host localhost --user root --password demo123 \
  --database ecommerce --port 3306 \
  "SELECT u.username, COUNT(o.id) as orders, ROUND(SUM(o.total_amount),2) as total_spent
   FROM users u LEFT JOIN orders o ON u.id = o.user_id 
   GROUP BY u.id ORDER BY total_spent DESC LIMIT 5"
```

**Clean up demo:**
```bash
docker stop sqleton-demo && docker rm sqleton-demo
```

## Connection Examples

### Direct Connection Flags
```bash
# MySQL
sqleton query --db-type mysql --host localhost --user root --password mypass \
  --database mydb --port 3306 "SELECT * FROM users"

# PostgreSQL  
sqleton query --db-type postgres --host localhost --user postgres \
  --database analytics --port 5432 "SELECT COUNT(*) FROM events"

# SQLite
sqleton query --db-type sqlite --database ./local.db "SELECT * FROM logs"

# DuckDB (in-memory engine reading local files)
sqleton query --db-type duckdb --database '' \
  "SELECT * FROM read_csv_auto('./exports/*.csv') LIMIT 10"

# DuckDB (persistent database file)
sqleton query --db-type duckdb --database ./analytics.duckdb \
  "SELECT * FROM my_cached_table LIMIT 10"

# DuckDB (URI-style DSN)
sqleton query --driver duckdb --dsn 'duckdb:///tmp/app.db' \
  "SELECT * FROM my_cached_table LIMIT 10"
```

### Environment Variables
```bash
export SQLETON_DB_TYPE=mysql
export SQLETON_HOST=localhost
export SQLETON_USER=root
export SQLETON_PASSWORD=mypass
export SQLETON_DATABASE=mydb

sqleton query "SELECT * FROM users WHERE created_at > '2024-01-01'"
```

### Application Configuration
Sqleton uses layered **app config** for repository discovery.

Supported locations are:
- `/etc/sqleton/config.yaml`
- `~/.sqleton/config.yaml`
- `$XDG_CONFIG_HOME/sqleton/config.yaml`
- `.sqleton.yml` at the git repository root
- `.sqleton.yml` in the current working directory

The preferred schema is:

```yaml
app:
  repositories:
    - /Users/manuel/code/ttc/ttc-dbt/sqleton-queries
    - /Users/manuel/.sqleton/queries
```

Repository lists merge in layer order, then `SQLETON_REPOSITORIES` is appended,
then the default `$HOME/.sqleton/queries` directory is added when it exists.
Legacy top-level `repositories:` is still accepted during the migration, but new
config should use `app.repositories`.

You can also add repository roots temporarily with:
```bash
export SQLETON_REPOSITORIES=/path/to/repo-a:/path/to/repo-b
```

A common setup is:

- global `~/.sqleton/config.yaml` for shared repositories
- project-local `.sqleton.yml` for project repositories
- explicit `--config-file` for database settings

### Explicit Command Configuration
Use `--config-file` when you want to load command-section settings such as
`sql-connection` or `dbt`. Command config remains explicit; sqleton does not
auto-discover database settings from home or project files.

```yaml
sql-connection:
  db-type: mysql
  host: localhost
  user: root
  password: mypass
  database: mydb
  port: 3306
```

Then run:
```bash
sqleton query --config-file ./db-config.yaml "SELECT COUNT(*) FROM users"
```

You can also point `--config-file` at a project `.sqleton.yml` if that file
contains command sections such as `sql-connection`; sqleton will only read the
known command sections during command parsing.

### DBT Profiles Integration
If you use dbt, sqleton can read your existing profiles:
```bash
# Use default dbt profile
sqleton query --use-dbt-profile "SELECT * FROM dim_customers"

# Use specific profile and target
sqleton query --dbt-profile analytics --dbt-target dev \
  "SELECT * FROM fact_orders WHERE order_date >= '2024-01-01'"
```

## Command Usage

### Basic Query Execution
```bash
# Execute SQL from command line
sqleton query "SELECT * FROM users WHERE active = true"

# Execute SQL from file
sqleton run queries/user-analysis.sql

# Execute SQL from stdin
cat report.sql | sqleton run -
```

### Query files directly with DuckDB
```bash
# Query a set of JSON arrays directly
sqleton query --db-type duckdb --database '' \
  "SELECT user_id, SUM(amount) AS total_amount
   FROM read_json_auto('./events/*.json', format='array')
   GROUP BY user_id"

# Query a set of CSV files directly
sqleton query --db-type duckdb --database '' \
  "SELECT region, SUM(revenue) AS total_revenue
   FROM read_csv_auto('./reports/*.csv')
   GROUP BY region"

# Query a parquet file directly
sqleton query --db-type duckdb --database '' \
  "SELECT product, SUM(amount) AS revenue
   FROM read_parquet('./warehouse/sales.parquet')
   GROUP BY product"
```

In these examples, the DuckDB connection itself is the sqleton database connection, while the file paths are passed inside the SQL through DuckDB functions such as `read_json_auto`, `read_csv_auto`, and `read_parquet`.

### Built-in Commands
```bash
# Test database connection
sqleton db test

# List available commands
sqleton commands list

# Get help for specific command
sqleton help database-sources
```

### Advanced SQL Command Files
```bash
# Run custom command with parameters
sqleton user-report --limit 100 --min-orders 5 active premium

# List available custom commands  
sqleton commands list --fields name,source

# Run a command from a local SQL command file
sqleton run-command ~/.sqleton/queries/user-stats.sql -- \
  --db-type sqlite --database ./local.db
```

## Output Customization

### Format Options
```bash
# Table format (default)
sqleton query "SELECT * FROM users" --output table

# JSON for API integration
sqleton query "SELECT * FROM users" --output json

# CSV for spreadsheets
sqleton query "SELECT * FROM users" --output csv

# YAML for configuration
sqleton query "SELECT * FROM users" --output yaml

# Custom Go template
sqleton query "SELECT name, email FROM users" \
  --template "{{.name}} <{{.email}}>"
```

### Field Selection and Filtering
```bash
# Select specific columns
sqleton query "SELECT * FROM users" --fields name,email,created_at

# Filter rows after query
sqleton query "SELECT * FROM products" --filter "price > 100"

# Combine filtering and field selection
sqleton query "SELECT * FROM orders" \
  --filter "status = 'completed'" \
  --fields order_id,customer_name,total
```

## Use Cases

### Development and Debugging
- **Quick data inspection**: Rapidly check database state during development
- **Query prototyping**: Test complex queries before putting them in application code
- **Database exploration**: Understand schema and data relationships
- **Performance testing**: Analyze query execution plans and timing

### Business Intelligence and Analytics
- **Customer analysis**: Segment users, analyze behavior patterns
- **Revenue reporting**: Generate daily, weekly, monthly financial reports
- **Inventory management**: Track stock levels, identify trending products
- **Operational metrics**: Monitor application performance and usage statistics

### Data Export and Integration
- **CSV generation**: Create spreadsheet-compatible reports for stakeholders
- **JSON APIs**: Generate data feeds for web applications and microservices  
- **Data pipeline**: Extract data for ETL processes and data warehousing
- **Report automation**: Schedule automated report generation and distribution
- **Local lakehouse exploration**: Use DuckDB to inspect JSON, CSV, and Parquet files directly from sqleton

### Database Administration
- **Schema inspection**: Quickly examine table structures and relationships
- **Data validation**: Verify data integrity and consistency across tables
- **Performance monitoring**: Track query performance and identify bottlenecks
- **Migration support**: Validate data before and after schema changes

## Advanced Features

### Command Repositories
Share and version-control your SQL commands by storing `.sql` command files in a
local repository directory such as `~/.sqleton/queries` or another configured
path. Store aliases next to them as `.alias.yaml` files.

### Template Functions
Powerful templating with SQL-specific helpers:

```sql
/* sqleton
name: recent-users
short: Users created after a given date
flags:
  - name: start_date
    type: date
  - name: status
    type: stringList
  - name: email_domain
    type: string
*/
SELECT * FROM users
WHERE created_at >= {{ .start_date | sqlDate }}
{{ if .status -}}
AND status IN ({{ .status | sqlStringIn }})
{{- end }}
{{ if .email_domain -}}
  AND email LIKE {{ .email_domain | sqlLike }}
  {{- end }}
```

### Server Mode
Run sqleton as a web service:

```bash
# Start HTTP server
sqleton serve --port 8080

# Execute queries via HTTP
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT COUNT(*) FROM users", "format": "json"}'
```

## Documentation

For detailed guides and references:

```bash
# Browse all help topics
sqleton help

# Specific topics
sqleton help database-sources
sqleton help duckdb-file-queries
sqleton help aliases  
sqleton help query-commands
sqleton help print-settings
```

**Online Documentation:**
- [Database Connection Guide](cmd/sqleton/doc/topics/02-database-sources.md)
- [DuckDB File Query Guide](cmd/sqleton/doc/topics/07-duckdb-file-queries.md)
- [SQL Command File Reference](cmd/sqleton/doc/topics/06-query-commands.md)
- [Output Format Options](cmd/sqleton/doc/topics/05-print-settings.md)
- [Examples and Tutorials](cmd/sqleton/doc/examples/)

## Why Choose sqleton?

**🚀 Speed**: Faster than setting up GUI clients or writing custom scripts
**🔧 Flexibility**: Works with multiple databases and output formats  
**📊 Professional**: Clean formatting suitable for presentations and reports
**🔄 Integration**: Seamless integration with existing tools and workflows
**📚 Powerful**: Handles complex queries while remaining simple to use
**🏗️ Extensible**: Custom commands and repositories for team collaboration

sqleton bridges the gap between simple command-line database clients and complex business intelligence tools, providing the perfect balance of power and simplicity for modern data workflows.

## License

sqleton is released under the MIT License. See [LICENSE](LICENSE) for details.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

**Built with ❤️ by the [go-go-golems](https://github.com/go-go-golems) team**
