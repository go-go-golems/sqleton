Output doc/demos/02-output-formats.gif

# Multiple Output Formats (JSON, CSV)
# This demo showcases sqleton's flexible output formats for different use cases

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set TypingSpeed 50ms
Set Theme "Dracula"

# Show the title
Type "# sqleton Demo: Multiple Output Formats"
Enter
Sleep 1s

# Set up alias
Type `alias sql='./sqleton query --db-type mysql --host localhost --user root --password demo123 --database ecommerce --port 3306'`
Enter
Sleep 1s

# JSON output for APIs
Type "# Generate JSON for API integration"
Enter
Type `sql --output json 'SELECT category, COUNT(*) as products, ROUND(AVG(price),2) as avg_price FROM products GROUP BY category LIMIT 3'`
Enter
Sleep 4s

Type ""
Enter
Type "# CSV output for Excel/BI tools"
Enter
Type `sql --output csv 'SELECT DATE(order_date) as day, COUNT(*) as orders, SUM(total_amount) as revenue FROM orders GROUP BY DATE(order_date) ORDER BY day DESC LIMIT 4'`
Enter
Sleep 4s

Type ""
Enter
Type "# YAML output for configuration"
Enter
Type `sql --output yaml 'SELECT username, email, status FROM users WHERE status = "active" LIMIT 3'`
Enter
Sleep 4s

Type ""
Enter
Type "# Perfect for data pipelines, reporting, and integration workflows"
Enter
Sleep 2s
