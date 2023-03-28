---
Title: Use subqueries in your SQL
Slug: subqueries
Short: |
  ```
  You can add subQueries as a field in your YAML sqleton command. They can then be used
  withing your templates to evaluate nested values.
  ```
Topics:
- queries
IsTemplate: false
IsTopLevel: false
ShowPerDefault: false
SectionType: GeneralTopic
---
If you need to use subqueries in your SQL, you can add them as a field in your YAML sqleton command.

They can then be used withing your templates to evaluate nested values.

This makes it easy to create pivoted tables, for example.

```yaml
name: posts-counts
short: Count posts by type
flags:
  - name: post_type
    description: Post type
    type: stringList
    required: false
subqueries:
  post_types: |
    SELECT DISTINCT post_type 
    FROM wp_posts
    GROUP BY post_type
    LIMIT 4
query: |
  {{ $types := sqlColumn (subQuery "post_types") }}
  SELECT
  {{ range $i, $v := $types }}
    {{ $v2 := printf "count%d" $i }}
    (
      SELECT count(*) AS count
      FROM wp_posts
      WHERE post_status = 'publish'
      AND post_type = '{{ $v }}'
    ) AS `{{ $v }}` {{ if not (eq $i (sub (len $types) 1)) }}, {{ end }}
  {{ end }}
```

If you print out the query before running it:

```
❯ sqleton wp posts-counts --print-query      
SELECT
   (
    SELECT count(*) AS count
    FROM wp_posts
    WHERE post_status = 'publish'
    AND post_type = 'attachment'
  ) AS `attachment` , 
  (
    SELECT count(*) AS count
    FROM wp_posts
    WHERE post_status = 'publish'
    AND post_type = 'custom_css'
  ) AS `custom_css` , 
  (
    SELECT count(*) AS count
    FROM wp_posts
    WHERE post_status = 'publish'
    AND post_type = 'export_template'
  ) AS `export_template` , 
  (
    SELECT count(*) AS count
    FROM wp_posts
    WHERE post_status = 'publish'
    AND post_type = 'faq'
  ) AS `faq` 
```

And then run it:

```
❯ sqleton wp posts-counts              
+------------+------------+-----------------+-----+
| attachment | custom_css | export_template | faq |
+------------+------------+-----------------+-----+
| 0          | 1          | 2               | 35  |
+------------+------------+-----------------+-----+
```