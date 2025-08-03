Output doc/demos/03-database-commands.gif

# Built-in Database Commands
# This demo shows sqleton's convenient database exploration commands

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set TypingSpeed 50ms
Set Theme "Dracula"

# Show the title
Type "# sqleton Demo: Built-in Database Commands"
Enter
Sleep 1s

# Test connection
Type "# Test database connection"
Enter
Type `./sqleton db test --db-type mysql --host localhost --user root --password demo123 --database ecommerce --port 3306`
Enter
Sleep 3s

# List tables
Type ""
Enter
Type "# List all tables in database"
Enter
Type `./sqleton db ls-tables --db-type mysql --host localhost --user root --password demo123 --database ecommerce --port 3306`
Enter
Sleep 3s

# Quick data inspection with select command
Type ""
Enter
Type "# Quick table inspection with built-in select"
Enter
Type `./sqleton select --table users --columns username,email,status --db-type mysql --host localhost --user root --password demo123 --database ecommerce --port 3306 --limit 4`
Enter
Sleep 4s

Type ""
Enter
Type "# No SQL needed for common database operations!"
Enter
Sleep 2s
