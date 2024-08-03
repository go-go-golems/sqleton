---
Title: Get the list of currently connected MySQL users
Slug: mysql-distinct-connected-users
Short: Show the list of connected users 
Topics:
- sqleton
- mysql
Commands:
- select
Flags:
- distinct
- columns
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Application
---

You can get the list of currently connected users on a MySQL database with:

``` 
❯ sqleton select --table information_schema.processlist --distinct --columns USER
+-------------------+
| USER              |
+-------------------+
| ttc_analytics_dev |
+-------------------+
```