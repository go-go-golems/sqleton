Output doc/demos/01-basic-query.txt

# Basic Query Execution with Table Output
# This demo shows sqleton's clean table output format for basic database queries

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set TypingSpeed 50ms
Set Theme "Dracula"

# Show the title
Type "# sqleton Demo: Basic Query Execution"
Enter
Sleep 1s

# Set up alias for cleaner commands
Type "# Set up convenient alias"
Enter
Type `alias sql='./sqleton query --db-type mysql --host localhost --user root --password demo123 --database ecommerce --port 3306'`
Enter
Sleep 1s

# Execute basic user query
Type "# Show active users with clean table formatting"
Enter
Type `sql 'SELECT id, username, email, status FROM users WHERE status = "active" LIMIT 5'`
Enter
Sleep 3s

# Show orders with better column display
Type ""
Enter
Type "# Check recent orders with formatted output"
Enter
Type `sql 'SELECT id, order_date, total_amount, status FROM orders ORDER BY order_date DESC LIMIT 4'`
Enter
Sleep 3s

# Show product analysis
Type ""
Enter  
Type "# Product inventory summary"
Enter
Type `sql 'SELECT category, COUNT(*) as products, AVG(price) as avg_price FROM products GROUP BY category ORDER BY avg_price DESC'`
Enter
Sleep 3s

Type ""
Enter
Type "# sqleton provides clean, professional table output perfect for terminal inspection"
Enter
Sleep 2s
