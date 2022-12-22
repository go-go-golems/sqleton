Output doc/gifs/demo.gif

Set FontSize 14
Set Width 1040
Set Height 1040

Set TypingSpeed 0.01

Type "sqleton db ls --use-dbt-profiles --fields name,type,hostname" Enter
Sleep 1200ms

Type "sqleton run examples/show-processlist.sql --output yaml" Enter
Sleep 2000ms

Type "sqleton mysql ps --select-template '{{.Id}} -- {{.User}}'" Enter
Sleep 1200ms

Type "sqleton wp ls-posts --fields ID,post_title" Enter
Sleep 1200ms

Type "sqleton help wp --examples"  Enter
Sleep 3000ms

Type "sqleton wp ls-posts --limit 100 --status publish --order-by post_title \" Enter
Type "    --from 2017-01-01 --to 2017-10-01 \" Enter
Type "    --fields ID,post_title,post_date \" Enter
Type "    --title-like Shrubs" Enter
Sleep 2000ms

Type "cat ~/.sqleton/queries/ttc/01-orders.yaml" Enter
Sleep 1200ms

Type "sqleton ttc orders" Enter
Sleep 1200ms

Type "sqleton ttc ls-orders --limit 10 --from 'last year' --to 'today' --print-query" Enter
Sleep 1200ms

Type "sqleton ttc ls-orders --limit 10 --from 'last year' --to 'today' --fields ID,post_date,post_status " Enter
Sleep 1200ms

Sleep 2000ms

