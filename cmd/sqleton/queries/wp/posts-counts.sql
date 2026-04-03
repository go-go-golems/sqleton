/* sqleton
name: posts-counts
short: Count posts by type
flags:
  - name: post_type
    help: Post type
    type: stringList
    required: false
*/
SELECT
  post_type,
  COUNT(*) AS post_count
FROM wp_posts
WHERE post_status = 'publish'
{{ if .post_type }}
  AND post_type IN ({{ .post_type | sqlStringIn }})
{{ end }}
GROUP BY post_type
ORDER BY post_count DESC, post_type ASC
