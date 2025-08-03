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
```

## Live Demos

See sqleton in action with these interactive demos using real ecommerce data:

### Basic Query Execution
![Basic Query Demo](doc/demos/01-basic-query.gif)

Clean, professional table output perfect for terminal inspection and development workflows.

### Multiple Output Formats
![Output Formats Demo](doc/demos/02-output-formats.gif)

JSON for APIs, CSV for Excel, YAML for configuration - sqleton adapts to your workflow.

### Built-in Database Commands
![Database Commands Demo](doc/demos/03-database-commands.gif)

Explore your database without writing SQL - test connections, list tables, and inspect data with built-in commands.

### Business Analytics & Complex Queries
![Business Analytics Demo](doc/demos/04-business-analytics.gif)

Handle complex joins, aggregations, and business intelligence queries with ease.

## Core Features

### Multiple Database Support
Connect to MySQL, PostgreSQL, and SQLite databases with consistent interface and authentication options.

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

### Using Homebrew
```bash
brew tap go-go-golems/go-go-go
brew install go-go-golems/go-go-go/sqleton
```

### Using Go
```bash
go install github.com/go-go-golems/sqleton/cmd/sqleton@latest
```

### From Source
```bash
git clone https://github.com/go-go-golems/sqleton
cd sqleton
go build ./cmd/sqleton
```

### Download Binaries
Get pre-built binaries from [GitHub Releases](https://github.com/go-go-golems/sqleton/releases)

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

### Configuration File
Create `~/.sqleton/config.yaml`:
```yaml
database:
  type: mysql
  host: localhost
  user: root
  password: mypass
  database: mydb
  port: 3306
```

Then use without connection flags:
```bash
sqleton query "SELECT COUNT(*) FROM users"
```

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

### Built-in Commands
```bash
# Test database connection
sqleton db test

# List available commands
sqleton queries

# Get help for specific command
sqleton help database-sources
```

### Advanced YAML Commands
```bash
# Run custom command with parameters
sqleton user-report --limit 100 --min-orders 5 active premium

# List available custom commands  
sqleton queries --fields name,source

# Run command from external repository
sqleton run-command https://github.com/myorg/sql-commands/user-stats.yaml
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

### Database Administration
- **Schema inspection**: Quickly examine table structures and relationships
- **Data validation**: Verify data integrity and consistency across tables
- **Performance monitoring**: Track query performance and identify bottlenecks
- **Migration support**: Validate data before and after schema changes

## Advanced Features

### Command Repositories
Share and version-control your SQL commands:

```bash
# Load commands from repository
sqleton queries --repository https://github.com/myorg/analytics-queries

# Use repository command
sqleton customer-lifetime-value --segment premium --period 2024
```

### Template Functions
Powerful templating with SQL-specific helpers:

```yaml
query: |
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
sqleton help aliases  
sqleton help query-commands
sqleton help print-settings
```

**Online Documentation:**
- [Database Connection Guide](cmd/sqleton/doc/topics/02-database-sources.md)
- [YAML Command Reference](cmd/sqleton/doc/topics/06-query-commands.md)
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
