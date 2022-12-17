---
Title: Show the list of all dbt profiles
Slug: ls-dbt-profiles
Short: |
  ```
  sqliton db ls --use-dbt-profiles
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
You can ask sqliton to list all dbt profiles it is able to find.

Don't forget to enable `--use-dbt-profiles`. Use `--dbt-profiles-path` to use another file.

---

```
❯ sqliton db ls --use-dbt-profiles
+-------------------+-----------+-------+-----------+-------+-------------------+-------------------+
| database          | name      | type  | hostname  | port  | username          | schema            |
+-------------------+-----------+-------+-----------+-------+-------------------+-------------------+
| ttc_analytics     | localhost | mysql | localhost | 3336  | root              | ttc_analytics     |
| ttc_analytics     | prod      | mysql | localhost | 50393 | ttc_analytics     | ttc_analytics     |
| ttc_analytics     | prod      | mysql | localhost | 50393 | ttc_analytics     | ttc_analytics     |
| ttc_dev_analytics | dev       | mysql | localhost | 50392 | ttc_dev_analytics | ttc_dev_analytics |
+-------------------+-----------+-------+-----------+-------+-------------------+-------------------+
```
