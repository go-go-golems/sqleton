---
Title: Show all posts about shrubs from 2017
Slug: wp-ls-posts-shrubs
Short: |
  ```
  sqleton wp ls-posts   
    --limit 100 --status publish --order-by post_title \
    --from 2017-01-01 --to 2017-10-01 --title-like Shrubs
  ```
Topics:
- wordpress
Commands:
- wp
- ls-posts
Flags:
- status
- order-by
- limit
- from
- to
- title-like
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Example
---

We can run much more complex queries against the Wordpress DB.

```
❯ sqleton wp ls-posts --limit 100 --status publish --order-by post_title \
   --from 2017-01-01 --to 2017-10-01 \
   --fields ID,post_title,post_date \
   --title-like Shrubs
+-------+--------------------------------------------------+---------------------+
| ID    | post_title                                       | post_date           |
+-------+--------------------------------------------------+---------------------+
| 15994 | Deer Resistant Trees and Shrubs                  | 2017-02-26 12:47:57 |
| 15722 | Flowering Trees and Shrubs for Early Spring      | 2017-02-07 11:08:15 |
| 23006 | Time to Prune Spring-flowering Shrubs - Part One | 2017-06-26 06:28:41 |
| 23259 | Time to Prune Spring-flowering Shrubs – Part Two | 2017-07-03 06:50:48 |
| 23520 | Tips on Placing Shrubs in Your Garden            | 2017-07-10 06:32:17 |
| 17662 | Tips on Pruning Shrubs and Flowering Trees       | 2017-04-09 06:35:22 |
| 24391 | Top Drought Resistant Trees and Shrubs           | 2017-08-08 03:44:00 |
| 16177 | What Shrubs Should be Pruned in Spring?          | 2017-03-05 07:48:52 |
+-------+--------------------------------------------------+---------------------+
```

```
❯ sqleton wp ls-posts   --limit 100 --status publish \
    --order-by post_title \
    --from 2017-01-01 --to 2017-10-01 \
    --title-like Shrubs --print-query
SELECT wp.ID, wp.post_title, wp.post_type, wp.post_status, wp.post_date FROM wp_posts wp
WHERE post_type IN ('post','page')
AND post_status IN ('publish')
AND post_date >= '2017-01-01'
AND post_date <= '2017-10-01'
AND post_title LIKE '%Shrubs%'
ORDER BY post_title
LIMIT 100
```
