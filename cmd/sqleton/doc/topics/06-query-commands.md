---
Title: Adding query commands
Slug: query-commands
Short: |
   You can add commands to the `sqleton` program in a variety of ways:
   - using `.sql` files with a sqleton metadata preamble
   - using Markdown files
Topics:
- queries
Commands:
- queries
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

## Using SQL files

SQL command files are regular `.sql` files with a sqleton-only preamble stored in
an opening block comment. SQL engines ignore the comment, while sqleton parses the
metadata before executing the remaining SQL body.

```sql
/* sqleton
name: ls-posts-type
short: Show all WP posts, limited, by type
long: Show all posts and their ID
flags:
   - name: types
     type: stringList
     default:
        - post
        - page
     help: Select posts by type
arguments:
   - name: limit
     shortFlag: l
     type: int
     default: 10
     help: Limit the number of posts
*/
SELECT wp.ID, wp.post_title, wp.post_type, wp.post_status FROM wp_posts wp
WHERE post_type IN ({{ .types | sqlStringIn }})
LIMIT {{ .limit }}
```

## Query repository

These files can be stored in a repository directory that has the following format:

``` 
repository/
   subCommand/
      subsubsCommand/
         query.sql
   subCommand2/
      query2.sql
```

This will result in the following commands being added (including their subcommands):

```
sqleton subCommand subsubsCommand query
sqleton subCommand2 query2
```

A repository can be loaded from an embedded query tree or from a filesystem
directory. In normal sqleton usage, query repositories are discovered from
layered application config and environment.

The repository-discovery locations are:
- `/etc/sqleton/config.yaml`
- `~/.sqleton/config.yaml`
- `$XDG_CONFIG_HOME/sqleton/config.yaml`
- `.sqleton.yml` at the git repository root
- `.sqleton.yml` in the current working directory

By default, queries in `$HOME/.sqleton/queries` are also loaded when that
directory exists.

The preferred app-owned schema is:

```yaml
app:
  repositories:
    - /Users/manuel/code/ttc/ttc-dbt/sqleton-queries
    - .sqleton/queries
```

Repository lists merge across layers in that order. You can also add
repositories temporarily with the `SQLETON_REPOSITORIES` environment variable.
It uses the normal OS path-list separator, so on Unix-like systems it looks
like:

```bash
export SQLETON_REPOSITORIES=/path/to/repo-a:/path/to/repo-b
```

Legacy top-level `repositories:` is no longer accepted. Move repository lists to
`app.repositories` instead. For a step-by-step migration guide, see:

- `sqleton help app-repositories-migration`

This application config is only for repository discovery. Command-section config
such as `sql-connection` or `dbt` should still be passed explicitly with
`--config-file`.

For example:

```yaml
sql-connection:
  db-type: sqlite
  database: ./local.db
```

```bash
sqleton run-command ./queries/ls-posts.sql -- --config-file ./db-config.yaml
```

A common workflow is:
- shared repositories in `~/.sqleton/config.yaml`
- project repositories in `.sqleton.yml`
- project database settings in an explicit command config file

## Using query parameters

Parameters are still declared in YAML, but only inside the sqleton preamble.
They are mapped to command-line flags and arguments.

Parameters have the following structure:

```yaml
- name: limit
  shortFlag: l
  type: int
  default: 10
  help: Limit the number of posts
```

Valid types for a parameter are:

- `string`
- `int`
- `bool`
- `date`
- `choice`
- `stringList`
- `intList`

These are then specified in the `flags` and `arguments` sections inside the
SQL preamble.

Arguments have to obey a few rules:
- optional arguments can't follow required arguments
- no argument can follow a stringList of intList argument


## Providing help pages for queries

To add examples, topics, and other help pages for your query, just add a markdown
file inside one of the directories scanned for help pages.

Look at [wordpress examples](../../../doc/examples/wp) for more examples.
