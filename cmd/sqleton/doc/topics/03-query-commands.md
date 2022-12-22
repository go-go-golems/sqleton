---
Title: Adding query commands
Slug: query-commands
Short: |
   You can add commands to the `sqleton` program in a variety of ways:
   - using YAML files 
   - using SQL files and metadata (TODO)
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

## Using YAML files

YAML files can be used to add commands to sqleton by using the following layout:

```yaml
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
query: |
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
         query.yaml
   subCommand2/
      query2.yaml
```

This will result in the following commands being added (including their subcommands):

```
sqleton subCommand subsubsCommand query
sqleton subCommand2 query2
```

A repository can be loaded at compile time as an `embed.FS` by using the
`sqleton.LoadSqlCommandsFromEmbedFS`, and at runtime from a directory by using
`sqleton.LoadSqlCommandsFromDirectory`.

The configuration flag or variable `repository` can be set to specify a custom
repository, by default, the queries in `$HOME/.sqleton/queries` are loaded.

## Using query parameters

A query can also provide parameters, which are mapped to command line flags and arguments

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

These are then specified in the `flags` and `arguments` section respectively.

Arguments have to obey a few rules:
- optional arguments can't follow required arguments
- no argument can follow a stringList of intList argument


## Providing help pages for queries

To add examples, topics, and other help pages for your query, just add a markdown
file inside one of the directories scanned for help pages.

Look at [wordpress examples](../examples/wp) for more examples.
