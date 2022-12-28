---
Title: Run a SQL query passed through stdin
Slug: run-stdin
Short: |
  ```
  echo "SHOW PROCESSLIST" | sqleton run --fields Id,User,Host -
  ```
Topics:
- mysql
Commands:
- run
IsTemplate: false
IsTopLevel: true
ShowPerDefault: false
SectionType: Example
---
You can pass queries into `sqleton run` by passing "-" as filename.

```
❯ echo "SHOW PROCESSLIST" | sqleton run --fields Id,User,Host -
+----------+-------------------+-------------------+
| Id       | User              | Host              |
+----------+-------------------+-------------------+
| 39636346 | ttc_analytics_dev | 172.31.20.6:40120 |
+----------+-------------------+-------------------+
```
