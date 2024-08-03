---
Title: Quickly create a query template
Slug: select-create-query
Short: |
  ```
  sqleton select --table orders \
       --columns order_number,status,totals \
       --limit 10 --order-by "order_number DESC" \
       --create-query orders
  ```
Topics:
- mysql
- queries
Commands:
- select
- queries
Flags:
- create-query
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Example
---
You can use `sqleton select` with the `--create-query <name>` flag to
quickly scaffold queries that can then be stored in `~/.sqleton/queries`.

``` 
❯ sqleton select --table orders --limit 20 \
     --order-by "order_number DESC" \
     --create-query orders 
name: orders
short: Select columns from orders
flags:
    - name: where
      type: string
    - name: limit
      type: int
      help: 'Limit the number of rows (default: 10), set to 0 to disable'
      default: 10
    - name: offset
      type: intG
      help: 'Offset the number of rows (default: 0)'
      default: 0
    - name: distinct
      type: bool
      help: 'Whether to select distinct rows (default: false)'
      default: false
    - name: order_by
      type: string
      help: 'Order by (default: order_number DESC)'
      default: order_number DESC
query: |-
    SELECT {{ if .distinct }}DISTINCT{{ end }} order_number, status, totals FROM orders
    {{ if .where  }}  WHERE {{.where}} {{ end }}
    {{ if .order_by }} ORDER BY {{ .order_by }}{{ end }}
    {{ if .limit }} LIMIT {{ .limit }}{{ end }}
    OFFSET {{ .offset }}
```

It will prepopulate most flags for the template from the values you pass it.
The flags `--columns` and `--where` however are fixed.

``` 
❯ sqleton select --table orders --create-query orders \
     --limit 50 --where "title LIKE '%anthropology%'"
name: orders
short: Select from orders where title LIKE '%anthropology%'
flags:
    - name: limit
      type: int
      help: 'Limit the number of rows (default: 50), set to 0 to disable'
      default: 50
    - name: offset
      type: int
      help: 'Offset the number of rows (default: 0)'
      default: 0
    - name: distinct
      type: bool
      help: 'Whether to select distinct rows (default: false)'
      default: false
    - name: order_by
      type: string
      help: Order by
query: |-
    SELECT {{ if .distinct }}DISTINCT{{ end }} * FROM orders WHERE title LIKE '%anthropology%'
    {{ if .order_by }} ORDER BY {{ .order_by }}{{ end }}
    {{ if .limit }} LIMIT {{ .limit }}{{ end }}
    OFFSET {{ .offset }}
```

The `--count` flag also severely restricts the number
of flags in the template:

```
❯ sqleton select --table orders --create-query orders --count --distinct --columns name
name: orders
short: Count all rows from orders
flags:
    - name: where
      type: string
query: |-
    SELECT COUNT(DISTINCT name) AS count FROM orders
    {{ if .where  }}  WHERE {{.where}} {{ end }}
    {{ if .order_by }} ORDER BY {{ .order_by }}{{ end }}
    {{ if .limit }} LIMIT {{ .limit }}{{ end }}
    OFFSET {{ .offset }}
}
```