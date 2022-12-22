---
Title: Get the ID of the last 5 draft posts for use in a shell script
Slug: wp-ls-posts-select
Short: |
  ```
  sqleton wp ls-posts --type blog --status draft --limit 5 --select ID
  ```
Topics:
- wordpress
Commands:
- ls-posts
- wp
Flags:
- status
- type
- limit
- select
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Example
---

To get only the IDs, to reuse them in another context (for example a shell script loop):

```
❯  sqleton wp ls-posts --type blog --status draft --limit 5 --select ID 
635239
76792
471151
262931
79536
```
