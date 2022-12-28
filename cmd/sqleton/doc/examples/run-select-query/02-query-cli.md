---
Title: Run a SQL query from the CLI
Slug: query-file
Short: |
  ```
  sqleton query "SHOW PROCESSLIST"
  ```
Topics:
- mysql
Commands:
- query
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Example
---
You can use sqleton to run a query straight from the CLI.
and use the full set of glazed flags.

```
❯ sqleton run examples/show-processlist.sql --fields User,Host,Command,Info
+-----------------+------------------+---------+------------------+
| User            | Host             | Command | Info             |
+-----------------+------------------+---------+------------------+
| event_scheduler | localhost        | Daemon  | <nil>            |
| ttc             | 172.20.0.7:41054 | Sleep   | <nil>            |
| ttc             | 172.20.0.7:41058 | Sleep   | <nil>            |
| root            | 172.20.0.1:61900 | Query   | SHOW PROCESSLIST |
+-----------------+------------------+---------+------------------+
```

```
❯ sqleton run examples/show-processlist.sql --select Id
4
29
30
549
```
