---
Title: Migrating repository config to app.repositories
Slug: sqleton-app-repositories-migration
Short: Migrate sqleton app config from legacy top-level repositories to app.repositories.
Topics:
- sqleton
- config
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Tutorial
---

## Goal

This tutorial shows how to migrate sqleton repository discovery config from the old top-level `repositories:` key to the current `app.repositories` block.

Use this guide if sqleton fails with an error like:

```text
legacy top-level repositories is no longer supported in /path/to/config.yaml; move entries to app.repositories
```

## What changed

Sqleton now treats repository discovery as **app-owned config**.

The supported shape is:

```yaml
app:
  repositories:
    - /path/to/repo-a
    - /path/to/repo-b
```

The old shape is no longer accepted:

```yaml
repositories:
  - /path/to/repo-a
  - /path/to/repo-b
```

## Why this changed

This makes sqleton's config model clearer:

- `app.repositories` is for repository discovery
- `--config-file` is for explicit command config such as `sql-connection` or `dbt`

That separation matters because repository discovery is layered automatically, while database settings remain explicit.

## Supported repository-discovery locations

Sqleton discovers repository config from these locations, in layer order:

- `/etc/sqleton/config.yaml`
- `~/.sqleton/config.yaml`
- `$XDG_CONFIG_HOME/sqleton/config.yaml`
- `.sqleton.yml` at the git repository root
- `.sqleton.yml` in the current working directory

Then sqleton appends:

- `SQLETON_REPOSITORIES`
- the default `$HOME/.sqleton/queries` directory when it exists

## Migration examples

### Example 1: global user config

Before:

```yaml
repositories:
  - ~/code/sql/shared
  - ~/.sqleton/queries-extra
```

After:

```yaml
app:
  repositories:
    - ~/code/sql/shared
    - ~/.sqleton/queries-extra
```

### Example 2: project-local config

Before `.sqleton.yml`:

```yaml
repositories:
  - ./queries
  - ../team-queries
```

After `.sqleton.yml`:

```yaml
app:
  repositories:
    - ./queries
    - ../team-queries
```

### Example 3: project repositories plus explicit DB config

Project-local `.sqleton.yml`:

```yaml
app:
  repositories:
    - ./queries
    - ../team-queries
```

Explicit DB config file:

```yaml
sql-connection:
  db-type: postgresql
  host: 127.0.0.1
  port: 5432
  database: app
  user: app
```

Run with:

```bash
sqleton run-command ./queries/list-users.sql -- --config-file ./db-config.yaml
```

## Can one file contain both app and command config?

Yes.

A `.sqleton.yml` file can contain both:

- `app.repositories`
- command sections like `sql-connection`

Example:

```yaml
app:
  repositories:
    - ./queries

sql-connection:
  db-type: sqlite
  database: ./local.db
```

Sqleton will:

- use `app.repositories` automatically for repository discovery
- only use `sql-connection` when you explicitly point `--config-file` at that file

For example:

```bash
sqleton query --config-file ./.sqleton.yml "SELECT COUNT(*) FROM users"
```

## Troubleshooting

### I updated my config but sqleton still does not find my query repository

Check:

- the file is in one of the supported locations
- the key is `app.repositories`, not `repositories`
- the paths are valid on your machine
- you did not accidentally put repository config only into a command config file that sqleton never auto-discovers

### I want project-local repositories and project-local DB settings

Use:

- `.sqleton.yml` for `app.repositories`
- either the same file or a second file for `sql-connection`
- `--config-file` when you want sqleton to read the command settings

## Summary

The migration rule is simple:

- move `repositories:` to `app.repositories`
- keep repository discovery in layered app config
- keep DB and dbt settings in explicit command config
