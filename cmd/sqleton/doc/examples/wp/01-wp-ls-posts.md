---
Title: Show last 10 posts in a wordpress database
Slug: wp-ls-posts
Short: |
  ```
  sqleton wp ls-posts   
  ```
Topics:
- wordpress
Commands:
- wp
- ls-posts
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Example
---
You can use sqleton to list the posts in a WordPress database.

Per default, we show only the last 10 posts and pages.

```
❯ sqleton wp ls-posts              
+--------+-------------------------------------------+-----------+-------------+---------------------+
| ID     | post_title                                | post_type | post_status | post_date           |
+--------+-------------------------------------------+-----------+-------------+---------------------+
| 703822 | Auto Draft                                | post      | auto-draft  | 2022-12-14 15:09:42 |
| 703808 | Thinking Hydrangeas? Think Native!        | post      | publish     | 2022-11-07 09:02:18 |
| 703466 | Winter Care of Houseplants                | post      | publish     | 2022-10-31 09:15:00 |
| 702794 | What is Wintergreen?                      | post      | publish     | 2022-10-24 21:06:00 |
| 702672 | What are Tetraploid Daylilies?            | post      | publish     | 2022-10-17 09:27:00 |
| 700051 | Growing Clematis – large-flowered hybrids | post      | publish     | 2022-10-10 09:53:00 |
| 698918 | Growing Sedges in Your Garden             | post      | publish     | 2022-10-03 09:41:00 |
| 697647 | Get to Know the Asters                    | post      | publish     | 2022-09-26 09:03:00 |
| 696470 | Why Isn’t My Garden Growing?              | post      | publish     | 2022-09-19 09:59:00 |
| 695361 | How Big Does That Tree Really Get?        | post      | publish     | 2022-09-12 09:36:00 |
+--------+-------------------------------------------+-----------+-------------+---------------------+
```

Looking at the query:

```
❯ sqleton wp ls-posts --print-query
SELECT wp.ID, wp.post_title, wp.post_type, wp.post_status, wp.post_date FROM wp_posts wp
WHERE post_type IN ('post','page')
ORDER BY post_date DESC
LIMIT 10
```