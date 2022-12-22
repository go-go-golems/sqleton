---
Title: Show last 10 posts in a wordpress database
Slug: wp-ls-posts
Short: |
  ```
  sqleton wp ls-posts --fields ID,post_title
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
❯ sqleton wp ls-posts  --fields ID,post_title
+--------+-------------------------------------------+
| ID     | post_title                                |
+--------+-------------------------------------------+
| 703822 | Auto Draft                                |
| 703808 | Thinking Hydrangeas? Think Native!        |
| 703466 | Winter Care of Houseplants                |
| 702794 | What is Wintergreen?                      |
| 702672 | What are Tetraploid Daylilies?            |
| 700051 | Growing Clematis – large-flowered hybrids |
| 698918 | Growing Sedges in Your Garden             |
| 697647 | Get to Know the Asters                    |
| 696470 | Why Isn’t My Garden Growing?              |
| 695361 | How Big Does That Tree Really Get?        |
+--------+-------------------------------------------+
```

Looking at the query:

```
❯ sqleton wp ls-posts --print-query
SELECT wp.ID, wp.post_title, wp.post_type, wp.post_status, wp.post_date FROM wp_posts wp
WHERE post_type IN ('post','page')
ORDER BY post_date DESC
LIMIT 10
```