---
Title: Show the process list of a mysql server
Slug: mysql-ps
Short: |
  ```
  sqleton mysql ps
  ```
Topics:
- mysql
Commands:
- mysql
- ps
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Example
---
You can use sqleton to run `show processlist` on a mysql server,
and use the full set of glazed flags.

This command bundles the actual query directly inside sqleton, instead of
loading it from a file as shown in `03-run-show-process-list`

```
❯ sqleton mysql ps --fields User,Host,Command,Info
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
❯ sqleton mysql ps --select Id
4
29
30
549
```

```
❯ sqleton mysql ps --select-template "{{.Id}} -- {{.User}}"
4 -- event_scheduler
4084 -- root
```
