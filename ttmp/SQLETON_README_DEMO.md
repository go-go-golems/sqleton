# ☠️ sqleton MySQL Demo Examples ☠️

This document showcases sqleton's core features using a realistic ecommerce database with MySQL.

## Setup Instructions

### 1. Start MySQL Server in Docker

```bash
# Start MySQL container with sample database
docker run --name mysql-demo \
  -e MYSQL_ROOT_PASSWORD=demo123 \
  -e MYSQL_DATABASE=ecommerce \
  -p 3306:3306 \
  -d mysql:8.0

# Wait for MySQL to start (about 30 seconds)
sleep 30
```

### 2. Create Sample Data

```sql
-- Create sample ecommerce tables with realistic data
USE ecommerce;

-- Users table with 8 sample users
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP NULL,
    status ENUM('active', 'inactive', 'suspended') DEFAULT 'active'
);

-- Products table with 12 sample products across multiple categories
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    category VARCHAR(50) NOT NULL,
    stock_quantity INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status ENUM('active', 'discontinued', 'out_of_stock') DEFAULT 'active'
);

-- Orders and order_items tables with realistic order history
-- [Full SQL schema included in mysql-sample-data.sql]
```

### 3. Set Connection Alias (Optional)

```bash
# Create alias for easier demo commands
alias sql='sqleton query --db-type mysql --host localhost --user root --password demo123 --database ecommerce --port 3306'
```

## Demo 1: Basic Query Execution

**Purpose**: Show simple SQL execution with table output (default format)

```bash
$ sqleton query --db-type mysql --host localhost --user root --password demo123 --database ecommerce --port 3306 'SELECT id, username, email, status FROM users LIMIT 5'
```

**Output**:
```
+----+------------+---------------------+----------+
| id | username   | email               | status   |
+----+------------+---------------------+----------+
| 1  | johndoe    | john@example.com    | active   |
| 2  | janesmit   | jane@example.com    | active   |
| 3  | bobwils    | bob@example.com     | active   |
| 4  | alicebrown | alice@example.com   | active   |
| 5  | charli_j   | charlie@example.com | inactive |
+----+------------+---------------------+----------+
```

**Key Value**: Clean, readable table output perfect for terminal use.

## Demo 2: Different Output Formats

**Purpose**: Demonstrate sqleton's flexible output formats for different use cases

### JSON Output (for APIs/integrations)
```bash
$ sql --output json 'SELECT name, price, category FROM products LIMIT 3'
```

**Output**:
```json
[
{
  "category": "Electronics",
  "name": "Wireless Headphones",
  "price": "129.99"
}
, {
  "category": "Appliances",
  "name": "Coffee Maker",
  "price": "89.99"
}
, {
  "category": "Sports",
  "name": "Running Shoes",
  "price": "149.99"
}
]
```

### CSV Output (for spreadsheets/reporting)
```bash
$ sql --output csv 'SELECT name, price, category FROM products LIMIT 3'
```

**Output**:
```csv
name,price,category
Wireless Headphones,129.99,Electronics
Coffee Maker,89.99,Appliances
Running Shoes,149.99,Sports
```

**Key Value**: Multiple output formats enable seamless integration with different tools and workflows.

## Demo 3: Advanced Queries and Aggregations

**Purpose**: Show sqleton handling complex SQL with aggregations and analytics

```bash
$ sql 'SELECT category, COUNT(*) as product_count, AVG(price) as avg_price, SUM(stock_quantity) as total_stock FROM products GROUP BY category ORDER BY avg_price DESC'
```

**Output**:
```
+-------------+---------------+------------+-------------+
| category    | product_count | avg_price  | total_stock |
+-------------+---------------+------------+-------------+
| Photography | 1             | 299.990000 | 3           |
| Outdoor     | 1             | 199.990000 | 12          |
| Kitchen     | 1             | 159.990000 | 18          |
| Sports      | 2             | 94.990000  | 156         |
| Appliances  | 1             | 89.990000  | 23          |
| Office      | 1             | 79.990000  | 34          |
| Electronics | 4             | 78.740000  | 221         |
| Food        | 1             | 49.990000  | 78          |
+-------------+---------------+------------+-------------+
```

**Key Value**: Handles complex SQL queries with grouping, aggregations, and calculations.

## Demo 4: Complex JOINs and Analytics

**Purpose**: Demonstrate sqleton with multi-table queries and business analytics

```bash
$ sql --output json 'SELECT u.username, COUNT(o.id) as order_count, ROUND(SUM(o.total_amount),2) as total_spent FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.id, u.username ORDER BY total_spent DESC LIMIT 5'
```

**Output**:
```json
[
{
  "order_count": 2,
  "total_spent": "389.97",
  "username": "johndoe"
}
, {
  "order_count": 1,
  "total_spent": "319.97",
  "username": "emilywils"
}
, {
  "order_count": 1,
  "total_spent": "199.99",
  "username": "susandav"
}
, {
  "order_count": 1,
  "total_spent": "179.98",
  "username": "bobwils"
}
, {
  "order_count": 2,
  "total_spent": "114.98",
  "username": "janesmit"
}
]
```

**Key Value**: Excellent for customer analytics, revenue analysis, and business intelligence queries.

## Demo 5: Built-in MySQL Commands

**Purpose**: Show sqleton's database introspection capabilities

### List Tables
```bash
$ sqleton mysql ls-tables --db-type mysql --host localhost --user root --password demo123 --database ecommerce --port 3306
```

**Output**:
```
+---------------------+
| Tables_in_ecommerce |
+---------------------+
| order_items         |
| orders              |
| products            |
| users               |
+---------------------+
```

### Select with Filtering
```bash
$ sqleton select --db-type mysql --host localhost --user root --password demo123 --database ecommerce --port 3306 --table products --where "category='Electronics'" --columns name,price,stock_quantity
```

**Output**:
```
+---------------------+--------+----------------+
| name                | price  | stock_quantity |
+---------------------+--------+----------------+
| Wireless Headphones | 129.99 | 45             |
| Smartphone Case     | 24.99  | 120            |
| Bluetooth Speaker   | 69.99  | 56             |
| Gaming Mouse        | 89.99  | 0              |
+---------------------+--------+----------------+
```

**Key Value**: Built-in commands for common database operations without writing SQL.

## Demo 6: Business Reporting

**Purpose**: Generate business reports suitable for export and analysis

```bash
$ sql --output csv 'SELECT DATE(o.order_date) as order_day, COUNT(*) as orders_count, ROUND(SUM(o.total_amount),2) as daily_revenue FROM orders o GROUP BY DATE(o.order_date) ORDER BY order_day'
```

**Output**:
```csv
order_day,orders_count,daily_revenue
2024-01-10,1,259.98
2024-01-11,1,89.99
2024-01-12,1,179.98
2024-01-13,1,129.99
2024-01-14,1,109.98
2024-01-15,1,199.99
2024-01-16,1,319.97
2024-01-17,1,24.99
```

**Key Value**: Perfect for generating daily/weekly/monthly business reports that can be imported into Excel or BI tools.

## Key Features Demonstrated

### 🎯 **Core Value Propositions**

1. **Multiple Output Formats**: Table, JSON, CSV - perfect for different use cases
2. **Complex Query Support**: Handles JOINs, aggregations, window functions, etc.
3. **Built-in Database Commands**: List tables, describe schema, select with filters
4. **Clean, Readable Output**: Professional formatting for both terminal and export
5. **Easy Connection Management**: Flexible authentication via flags or config
6. **Business Intelligence Ready**: Excellent for analytics and reporting workflows

### 🚀 **Use Cases Covered**

- **Developer Debugging**: Quick data inspection and validation
- **Business Analytics**: Customer analysis, revenue reporting, inventory tracking
- **Data Export**: CSV generation for spreadsheets and BI tools
- **API Integration**: JSON output for microservices and web applications
- **Database Administration**: Schema inspection and table management
- **Report Generation**: Automated business reporting workflows

### 💡 **Why Choose sqleton?**

- **Speed**: Faster than setting up database clients or writing custom scripts
- **Flexibility**: Works with multiple databases (MySQL, PostgreSQL, SQLite)
- **Integration**: Output formats integrate seamlessly with existing tools
- **Professional**: Clean formatting suitable for presentations and reports
- **Powerful**: Handles complex queries while remaining simple to use

---

## Database Schema Overview

The demo uses a realistic ecommerce database with these tables:

- **users** (8 records): Customer accounts with login tracking
- **products** (12 records): Product catalog across 8 categories
- **orders** (8 records): Customer orders with status tracking  
- **order_items** (16 records): Individual items within orders

This schema supports realistic business queries for customer analytics, inventory management, and sales reporting.

## Cleanup

```bash
# Stop and remove demo container
docker stop mysql-demo
docker rm mysql-demo
```
