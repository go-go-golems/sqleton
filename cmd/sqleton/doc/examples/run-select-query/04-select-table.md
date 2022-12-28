---
Title: Quickly select a table
Slug: select-table
Short: |
  ```
  sqleton select orders \
       --columns order_number,status,totals \
       --limit 10 --order-by "order_number DESC"
  ```
Topics:
- mysql
Commands:
- select
Flags:
- fields
- limit
- order-by
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Example
---
You can use sqleton to run a query straight from the CLI.
and use the full set of glazed flags.

``` 
❯ sqleton select orders \
       --columns order_number,status,totals \
       --limit 10 --order-by "order_number DESC"
+--------------+--------------+--------+
| status       | order_number | totals |
+--------------+--------------+--------+
| wc-completed | 8002         | 49.45  |
| wc-completed | 7968         | 395.00 |
| wc-completed | 7967         | 128.95 |
| wc-completed | 7966         | 88.95  |
| wc-completed | 7956         | 79.45  |
| wc-completed | 7954         | 136.69 |
| wc-cancelled | 7953         | 136.69 |
| wc-completed | 7944         | 108.95 |
| wc-completed | 7943         | 103.50 |
| wc-cancelled | 7937         | 157.50 |
+--------------+--------------+--------+
```
