---
Title: Show the list of all dbt profiles
Slug: ls-dbt-profiles
Short: |
  ```
  sqleton db ls --use-dbt-profiles
  ```
Topics:
- dbt
Commands:
- ls
Flags:
- use-dbt-profiles
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Example
---
You can ask sqleton to list all dbt profiles it is able to find.

Don't forget to enable `--use-dbt-profiles`. Use `--dbt-profiles-path` to use another file.

---

```
❯ sqleton db ls --use-dbt-profiles --fields name,type,hostname,database
+---------------------+-------+-----------+-------------------+
| name                | type  | hostname  | database          |
+---------------------+-------+-----------+-------------------+
| localhost.localhost | mysql | localhost | ttc_analytics     |
| ttc.prod            | mysql | localhost | ttc_analytics     |
| prod.prod           | mysql | localhost | ttc_analytics     |
| dev.dev             | mysql | localhost | ttc_dev_analytics |
+---------------------+-------+-----------+-------------------+
```
