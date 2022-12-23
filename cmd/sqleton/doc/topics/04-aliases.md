---
Title: Creating aliases
Slug: creating-aliases
Short: |
   You can add quickly create aliases for existing commands in order
   to save flags. These aliases can be stored alongside the query files
   and will become accessible as their own full-fledged cobra commands.
Topics:
- queries
- aliases
Commands:
- queries
Flags:
- create-alias
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

## Creating aliases

Due to the very flexible glazed output system and the flags available for each command,
it is useful to be able to create aliases to save on typing and having to remember
all variations. 

An alias is just a yaml file stored along the query yaml definition. Aliases can be 
located in an `embed.FS` repository, or in a filesystem repository. 

For a query `ttc/wp/posts.yaml`, aliases can be stored in the directory `ttc/wp/posts/`.

For example, we could have the following alias to show the newest drafts, 
in `newest-drafts.yaml`:

```yaml
name: newest-drafts
aliasFor: posts
flags:
  limit: 10
  from: last week
  status: draft
  filter: post_status,post_type
```

This command can then be executed by running `sqleton ttc wp posts newest-drafts`,
and will be equivalent to running 

```
sqleton ttc wp posts --limit 10 --from "last week" \
   --status draft --filter post_status,post_type
```

In order to help with the arduous task of writing YAML file, you can 
automatically emit the alias file by using the `--create-alias [name]` flag:

```
❯ sqleton wp postmeta --key-like 'refund_reason' --db ttc_prod --order-by "post_id DESC" --create-alias refunds                  
Flag db changed to ttc_prod
Flag key-like changed to refund_reason
Flag order-by changed to post_id DESC
name: refunds
aliasFor: postmeta
flags:
    db: ttc_prod
    key-like: refund_reason
    order-by: post_id DESC
```

