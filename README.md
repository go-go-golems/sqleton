# ☠️ sqleton ☠️ - a tool to quickly execute SQL commands

[![golangci-lint](https://github.com/wesen/sqleton/actions/workflows/lint.yml/badge.svg)](https://github.com/wesen/sqleton/actions/workflows/lint.yml)
[![golang-pipeline](https://github.com/wesen/sqleton/actions/workflows/push.yml/badge.svg)](https://github.com/wesen/sqleton/actions/workflows/push.yml)

![sqleton logo](doc/logo.png)

sqleton is a tool to easily run SQL commands.

## ☠️ Main features ☠️

- easily make a self-contained CLI to interact with your application
- extend the CLI application with your own repository of commands
- rich data format output thanks to [glazed](https://github.com/wesen/glazed)
- rich help system courtesy of [glazed](https://github.com/wesen/glazed)
    - easily add documentation to your CLI by editing markdown files
- support for connection information from:
    - environment variables
    - configuration file
    - command line flags
    - DBT profiles
- easy API for developers to supply custom commands

## Demo

![Demo of sqleton](https://imgur.com/a/H9tWd0h.gif)

## Overview

sqleton makes it easy to specify SQL commands along with parameters in a YAML file and run 
the result as a nice CLI application.

These files look like the following:

```yaml
name: ls-posts [types...]
short: "Show all WordPress posts"
long: |
  Show all WordPress posts and their ID, allowing you to filter by type, status,
  date and title.
flags:
  - name: limit
    shortFlag: l
    type: int
    default: 10
    help: Limit the number of posts
  - name: status
    type: stringList
    help: Select posts by status
    required: false
  - name: order_by
    type: string
    default: post_date DESC
    help: Order by column
  - name: types
    type: stringList
    default:
      - post
      - page
    help: Select posts by type
    required: false
  - name: from
    type: date
    help: Select posts from date
    required: false
  - name: to
    type: date
    help: Select posts to date
    required: false
  - name: title_like
    type: string
    help: Select posts by title
    required: false
query: |
  SELECT wp.ID, wp.post_title, wp.post_type, wp.post_status, wp.post_date FROM wp_posts wp
  WHERE post_type IN ({{ .types | sqlStringIn }})
  {{ if .status -}}
  AND post_status IN ({{ .status | sqlStringIn }})
  {{- end -}}
  {{ if .from -}}
  AND post_date >= {{ .from | sqlDate }}
  {{- end -}}
  {{ if .to -}}
  AND post_date <= {{ .to | sqlDate }}
  {{- end -}}
  {{ if .title_like -}}
  AND post_title LIKE {{ .title_like | sqlLike }}
  {{- end -}}
  ORDER BY {{ .order_by }}
  LIMIT {{ .limit }}
```

Furthermore, these commands can be included in the binary itself, which makes
distributing a rich set of queries very easy. The same concept is used to provide
a rich help system, which are just bundled markdown files. Instead of having to edit
the source code to provide advanced documentation for flags, commands and general topics,
all you need to do is add a markdown file in the doc/ directory!

```
❯ sqleton queries --fields name,source
+--------------------------+--------------------------------------------------------+
| name                     | source                                                 |
+--------------------------+--------------------------------------------------------+
| ls-posts                 | embed:queries/examples/01-get-posts.yaml               |
| ls-posts-limit           | embed:queries/examples/02-get-posts-limit.yaml         |
| ls-posts-type [types...] | embed:queries/examples/03-get-posts-by-type.yaml       |
| full-ps                  | embed:queries/mysql/01-show-full-processlist.yaml      |
| ls-tables                | embed:queries/mysql/02-show-tables.yaml                |
| ls-posts [types...]      | embed:queries/wp/ls-posts.yaml                         |
| count                    | file:/Users/manuel/.sqleton/queries/misc/count.yaml    |
| ls-orders                | file:/Users/manuel/.sqleton/queries/ttc/01-orders.yaml |
+--------------------------+--------------------------------------------------------+
```

It makes heavy use of my [glazed](https://github.com/wesen/glazed) library,
and in many ways is a test-driver for its development
