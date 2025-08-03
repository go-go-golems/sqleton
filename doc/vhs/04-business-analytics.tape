Output doc/demos/04-business-analytics.txt

# Complex Analytics & Business Queries
# This demo showcases sqleton for business intelligence and complex analysis

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set TypingSpeed 50ms
Set Theme "Dracula"

# Show the title
Type "# sqleton Demo: Business Analytics & Complex Queries"
Enter
Sleep 1s

# Set up alias
Type `alias sql='./sqleton query --db-type mysql --host localhost --user root --password demo123 --database ecommerce --port 3306'`
Enter
Sleep 1s

# Customer spending analysis
Type "# Customer spending analysis"
Enter
Type `sql 'SELECT u.username, COUNT(o.id) as orders, ROUND(SUM(o.total_amount),2) as total_spent FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.id ORDER BY total_spent DESC LIMIT 5'`
Enter
Sleep 4s

# Product performance with joins
Type ""
Enter
Type "# Product performance analysis"
Enter
Type `sql 'SELECT p.name, p.category, SUM(oi.quantity) as units_sold, ROUND(SUM(oi.quantity * oi.unit_price),2) as revenue FROM products p LEFT JOIN order_items oi ON p.id = oi.product_id GROUP BY p.id ORDER BY revenue DESC LIMIT 5'`
Enter
Sleep 4s

# Daily revenue trend
Type ""
Enter
Type "# Daily revenue trends"
Enter
Type `sql 'SELECT DATE(order_date) as day, COUNT(*) as orders, ROUND(SUM(total_amount),2) as revenue, ROUND(AVG(total_amount),2) as avg_order FROM orders GROUP BY DATE(order_date) ORDER BY day DESC'`
Enter
Sleep 4s

Type ""
Enter
Type "# Perfect for complex business intelligence and data analysis"
Enter
Sleep 2s
