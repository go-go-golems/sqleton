---
Title: SQL files with metadata preambles for sqleton design and implementation guide
Ticket: SQLETON-01-SQL-COMMAND-LOADER-REVIEW
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: clay/pkg/sql/query.go
      Note: Existing render and execution boundary that stays unchanged
    - Path: clay/pkg/sql/template.go
      Note: Existing SQL templating contract that the new source format should preserve
    - Path: glazed/pkg/cmds/alias/alias.go
      Note: Explicit alias file design should still compile to the existing alias runtime model
    - Path: go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md
      Note: Conceptual model for source scanning
    - Path: go-go-goja/pkg/jsverbs/scan.go
      Note: Frontmatter splitting and static source parsing inspiration
    - Path: sqleton/cmd/sqleton/cmds/select.go
      Note: Current create-query scaffolding path that should emit SQL in the new design
    - Path: sqleton/pkg/cmds/loaders.go
      Note: Current YAML parser baseline that the SQL parser should parallel
ExternalSources: []
Summary: Proposal for SQL-first sqleton command files using a top-of-file metadata preamble so command metadata stays structured while the main query remains plain SQL.
LastUpdated: 2026-04-02T08:50:58.148344659-04:00
WhatFor: Define a SQL-first command source format for sqleton where metadata lives in a preamble and the main query remains ordinary SQL.
WhenToUse: Read this when implementing or reviewing `.sql`-backed sqleton commands, migration away from YAML query bodies, or metadata parsing inspired by jsverbs frontmatter.
---



# SQL files with metadata preambles for sqleton design and implementation guide

## Executive Summary

The recommended new source format is:

- a normal `.sql` file,
- a top-of-file SQL comment block containing YAML metadata,
- and the main query body immediately below it as raw SQL text.

Example:

```sql
/* sqleton
name: ls-posts
short: Show all WordPress posts
tags:
  - wp
  - reporting
flags:
  - name: limit
    type: int
    default: 10
    help: Limit rows
  - name: types
    type: stringList
    default:
      - post
      - page
    help: Post types to include
long: |
  This command lists WordPress posts and basic metadata.
*/
SELECT
  wp.ID,
  wp.post_title AS title,
  wp.post_type AS type
FROM wp_posts wp
WHERE wp.post_type IN ({{ .types | sqlStringIn }})
ORDER BY wp.post_date DESC
LIMIT {{ .limit }}
```

This format keeps the best part of the current system, which is declarative metadata for flags, docs, tags, and layout, while fixing the worst authoring pain point, which is storing SQL inside a YAML text field.

The design is intentionally modeled after the successful separation in `go-go-goja` `jsverbs`:

- parse source metadata first,
- build a neutral model,
- then compile it into commands,
- and only later execute runtime behavior.

## Problem Statement

Today `sqleton` stores command metadata and SQL together in YAML files. That causes several practical problems for command authors:

- SQL syntax highlighting is worse because the query is inside `query: |`.
- Large query edits become noisy because indentation and YAML quoting are mixed with SQL changes.
- Long-form documentation and parameter metadata crowd the same YAML document as the SQL body.
- Tooling such as `select --create-query` naturally emits YAML, which reinforces the same storage constraint.

The desired end state is different:

- the command body should look like a real SQL file,
- metadata should be available at the top of the file,
- and the metadata should be ignored by SQL engines so the same file remains legible as SQL.

The user also explicitly asked to use `go-go-goja`'s `jsverbs` design for inspiration. That is the right model here because `jsverbs` treats source files as declarative input first and executable content second. The relevant references are:

- `go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md:29-33`
- `go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md:108-145`
- `go-go-goja/pkg/jsverbs/scan.go:567-572`
- `go-go-goja/pkg/jsverbs/scan.go:826-855`

The transferable lesson is not "make SQL look like JavaScript". The lesson is "separate source scanning from runtime execution and give metadata a clear static parsing stage."

## Proposed Solution

Use one-file SQL command sources with a structured preamble comment.

### Format overview

Each command file has two regions:

1. `preamble`
   a top-of-file SQL block comment containing YAML metadata
2. `body`
   the main query template as raw SQL

High-level grammar:

```text
sql-command-file
  := preamble body

preamble
  := "/*" SP? "sqleton" NEWLINE yaml-text "*/"

body
  := raw-sql-template
```

The parser ignores leading whitespace before the preamble, but the first meaningful token in the file must be the sqleton comment block. That keeps detection simple and deterministic.

### Why a block comment, not a line-comment frontmatter block

Both of these can work:

- prefix every metadata line with `--`
- put raw YAML inside `/* ... */`

The block-comment version is cleaner for authors:

- no extra prefix on every YAML line,
- easier multiline `long: |` or `flags:` indentation,
- easier to copy/paste from existing YAML specs,
- and easier for a parser to strip cleanly.

SQL engines treat the block as a comment. `sqleton` should strip the block before rendering or executing the body, so the metadata never reaches the database driver.

### Metadata schema

Reuse the current sqleton schema as much as possible so the source-format change is mostly about storage, not semantics.

Recommended metadata keys:

```yaml
name: ls-posts
short: Show all WordPress posts
long: |
  Longer help text rendered in CLI help output.
flags:
  - name: limit
    type: int
    default: 10
    help: Limit rows
arguments:
  - name: blog_id
    type: int
    help: Blog identifier
tags:
  - wp
  - reporting
layout: []
layers: []
metadata: {}
```

The main SQL body comes from the rest of the file after the closing `*/`.

### What happens to subqueries

This is the one place where the SQL-first format needs a deliberate product decision.

Recommendation:

- MVP:
  support one main SQL body only
- preferred authoring style:
  use CTEs or nested SQL directly in the main body
- phase 2 if needed:
  add explicit multi-fragment support, but do not block the MVP on that

Reasoning:

- the user's primary pain point is "SQL in YAML text fields"
- the main query body is the important win
- subquery section design can easily overcomplicate the first implementation

If named fragments are later required, keep them SQL-first as well. Do not reintroduce SQL text nested inside YAML metadata.

Possible phase-2 fragment syntax:

```sql
/* sqleton
name: post-counts
short: Count posts
*/
-- @sqleton:query main
SELECT ...

-- @sqleton:query post_types
SELECT DISTINCT post_type FROM wp_posts
```

That is intentionally not part of the MVP recommendation.

## Design Decisions

### Decision 1: keep one file per command

Rationale:

- command repositories are already filesystem-oriented,
- one file is easy to browse, diff, and move,
- and it preserves the current mental model from YAML sources.

### Decision 2: keep metadata YAML, but keep SQL out of YAML

Rationale:

- parameter definitions, tags, and help text are still structured metadata
- YAML remains convenient for those pieces
- the real pain is not metadata-in-YAML, it is SQL-in-YAML

### Decision 3: parse preamble before any runtime compilation

Rationale:

- this follows the `jsverbs` lesson directly,
- it creates one static source-scanning stage,
- and it keeps execution code ignorant of source-format details.

### Decision 4: compile `.sql` and `.yaml` through the same `SqlCommandSpec`

Rationale:

- one runtime model is easier to maintain,
- tests can be shared,
- and format choice becomes a source concern instead of a behavior fork.

### Decision 5: keep the SQL body templated exactly like today

Rationale:

- the goal is source ergonomics, not a templating rewrite,
- `clay/pkg/sql/template.go` and `clay/pkg/sql/query.go` already define the runtime contract,
- and changing both format and templating semantics at once would add needless risk.

### Decision 6: make aliases explicit instead of implicit fallbacks

Rationale:

- aliases are useful, but they should be a separate file kind
- command parse errors should remain command parse errors
- the `.sql` command format becomes much simpler if alias files are visibly different from command files

Recommended rule:

- `.sql` means command
- `.alias.yaml` or `.alias.yml` means alias
- no SQL command parse should ever fall back to alias parsing

## Intern-Oriented Architecture Walkthrough

Think about the future system as five stages.

```text
.sql file
  |
  v
split preamble/body
  |
  v
yaml decode preamble -> SqlCommandSpec
  |
  v
compile spec -> SqlCommand
  |
  v
repository + cobra registration
  |
  v
render query + execute query
```

### Stage 1: source splitting

Input:

- bytes from `queries/foo/bar.sql`

Output:

- `metadataText`
- `sqlBody`

This stage should not know anything about Cobra, parameter layers, or database connections. It only knows file syntax.

### Stage 2: metadata decoding

Input:

- `metadataText`

Output:

- `SqlCommandSpec`

This stage is responsible for:

- YAML decoding,
- schema validation,
- and setting `spec.Query = sqlBody`

### Stage 3: command compilation

Input:

- validated `SqlCommandSpec`

Output:

- `*sqleton_cmds.SqlCommand`

This stage applies shared layers and runtime wiring.

### Stage 4: repository registration

Input:

- compiled command
- path-derived parents from directory layout

Output:

- command inserted into the repository trie
- help docs loaded for the repository

### Stage 5: runtime execution

Input:

- parsed layers from Cobra/Glazed
- `SqlCommand.Query`
- database connection from `clay/pkg/sql/settings.go`

Output:

- rendered query string
- rows passed into the Glazed processor

This stage should not know or care whether the source file was YAML or SQL.

## Making Aliases Explicit

Aliases do not need to disappear, but they do need to stop complicating command parsing.

The simplest design is:

```text
queries/
  pg/
    connections.sql
    locks.sql
    aliases/
      active.alias.yaml
      admin.alias.yaml
```

Recommended alias file shape:

```yaml
name: active
aliasFor: connections
short: Show active connections
flags:
  state: active
```

Recommended dispatch rules:

```text
if path ends with ".alias.yaml" or ".alias.yml":
    parse as alias
else if path ends with ".sql":
    parse as sqleton SQL command
else if legacy yaml migration mode is enabled:
    parse as legacy YAML command
else:
    reject as unsupported
```

This is a real simplification, not just a naming preference.

Without explicit alias files:

- command parsing and alias parsing have to share a fallback path
- loader errors become less trustworthy
- the migration to `.sql` commands inherits old ambiguity

With explicit alias files:

- source kind is known from path before full parsing
- parser code is smaller and easier to test
- repository layout becomes self-documenting

If you want to simplify further later, aliases can even move into a separate mounted repository. That is optional. The main win comes from explicit source-kind separation, not from physically splitting repositories.

## Parser Contract

Suggested parser API:

```go
type SqlFilePreamble struct {
    Name      string                            `yaml:"name"`
    Short     string                            `yaml:"short"`
    Long      string                            `yaml:"long,omitempty"`
    Layout    []*layout.Section                 `yaml:"layout,omitempty"`
    Flags     []*parameters.ParameterDefinition `yaml:"flags,omitempty"`
    Arguments []*parameters.ParameterDefinition `yaml:"arguments,omitempty"`
    Layers    []layers.ParameterLayer           `yaml:"layers,omitempty"`
    Tags      []string                          `yaml:"tags,omitempty"`
    Metadata  map[string]interface{}            `yaml:"metadata,omitempty"`
}

func ParseSQLFileSpec(path string, contents []byte) (*SqlCommandSpec, error)
```

Recommended loader dispatch API:

```go
type SourceKind int

const (
    SourceUnknown SourceKind = iota
    SourceSQLCommand
    SourceYAMLCommand
    SourceYAMLAlias
)

func DetectSourceKind(path string, contents []byte) SourceKind
```

Suggested pseudocode:

```go
func ParseSQLFileSpec(path string, contents []byte) (*SqlCommandSpec, error) {
    metaText, body, err := splitSqletonSQLPreamble(contents)
    if err != nil {
        return nil, errors.Wrapf(err, "parse sqleton sql preamble: %s", path)
    }

    meta := &SqlFilePreamble{}
    if err := yaml.Unmarshal([]byte(metaText), meta); err != nil {
        return nil, errors.Wrapf(err, "decode sqleton sql metadata: %s", path)
    }

    spec := &SqlCommandSpec{
        Name:      meta.Name,
        Short:     meta.Short,
        Long:      meta.Long,
        Layout:    meta.Layout,
        Flags:     meta.Flags,
        Arguments: meta.Arguments,
        Layers:    meta.Layers,
        Tags:      meta.Tags,
        Metadata:  meta.Metadata,
        Query:     strings.TrimSpace(body),
        Format:    "sql",
    }

    if err := ValidateSqlCommandSpec(spec); err != nil {
        return nil, errors.Wrapf(err, "validate sqleton sql command: %s", path)
    }
    return spec, nil
}
```

Suggested preamble splitter:

```go
func splitSqletonSQLPreamble(contents []byte) (string, string, error) {
    s := strings.TrimLeft(string(contents), "\ufeff\r\n\t ")
    if !strings.HasPrefix(s, "/*") {
        return "", "", ErrMissingPreamble
    }

    end := strings.Index(s, "*/")
    if end == -1 {
        return "", "", ErrUnterminatedPreamble
    }

    raw := strings.TrimSpace(s[2:end])
    if !strings.HasPrefix(raw, "sqleton") {
        return "", "", ErrInvalidPreambleMarker
    }

    meta := strings.TrimSpace(strings.TrimPrefix(raw, "sqleton"))
    body := strings.TrimSpace(s[end+2:])
    return meta, body, nil
}
```

Suggested explicit dispatch pseudocode:

```go
func LoadSqletonEntry(path string, contents []byte) ([]cmds.Command, error) {
    switch DetectSourceKind(path, contents) {
    case SourceSQLCommand:
        spec, err := ParseSQLFileSpec(path, contents)
        if err != nil {
            return nil, err
        }
        cmd, err := compiler.Compile(spec)
        if err != nil {
            return nil, err
        }
        return []cmds.Command{cmd}, nil

    case SourceYAMLCommand:
        spec, err := ParseYAMLSqlCommandSpec(bytes.NewReader(contents))
        if err != nil {
            return nil, err
        }
        cmd, err := compiler.Compile(spec)
        if err != nil {
            return nil, err
        }
        return []cmds.Command{cmd}, nil

    case SourceYAMLAlias:
        a, err := alias.NewCommandAliasFromYAML(bytes.NewReader(contents))
        if err != nil {
            return nil, err
        }
        return []cmds.Command{a}, nil

    default:
        return nil, ErrUnsupportedSourceKind
    }
}
```

## Suggested Source Examples

### Example 1: simple command

```sql
/* sqleton
name: tables
short: Show SQLite tables
*/
SELECT name
FROM sqlite_master
WHERE type = 'table'
ORDER BY name
```

### Example 2: flags plus docs

```sql
/* sqleton
name: connections
short: Show current PostgreSQL connections
long: |
  Useful for quick operational debugging.
flags:
  - name: dbname
    type: string
    help: Database name to filter
  - name: state
    type: string
    help: Connection state to filter
  - name: limit
    type: int
    default: 50
    help: Maximum rows to return
tags:
  - pg
  - admin
*/
SELECT
  pid,
  usename AS user,
  datname AS dbname,
  state,
  query
FROM pg_stat_activity
WHERE 1=1
{{ if .dbname }} AND datname = {{ .dbname | sqlString }} {{ end }}
{{ if .state }} AND state = {{ .state | sqlString }} {{ end }}
LIMIT {{ .limit }}
```

### Example 3: generated output from `select --create-query`

The future `select --create-query` command should emit `.sql`, not YAML:

```sql
/* sqleton
name: orders
short: Select columns from orders
flags:
  - name: where
    type: stringList
  - name: limit
    type: int
    default: 100
    help: Limit the number of rows
*/
SELECT *
FROM orders
WHERE 1=1
{{ range .where }} AND {{ . }} {{ end }}
LIMIT {{ .limit }}
```

## Alternatives Considered

### Alternative 1: markdown files with fenced SQL blocks

This would allow rich documentation, but it makes the executable query body harder to extract and less natural to open in SQL tooling.

### Alternative 2: line-comment frontmatter

Example:

```sql
-- name: ls-posts
-- short: Show posts
-- ...
SELECT ...
```

This is workable, but block comments are cleaner for multiline YAML and more ergonomic to edit.

### Alternative 3: JSON metadata block

JSON is stricter and simpler to parse, but it is much less pleasant for human-edited flag definitions than YAML.

### Alternative 4: keep YAML forever and teach editors to cope

This addresses symptoms, not the underlying source-format problem.

## Implementation Plan

### Phase 1: internal refactor first

Before adding `.sql`, extract the common `SqlCommandSpec` pipeline described in the companion architecture review doc.

### Phase 2: add SQL parsing

1. Introduce a new `.sql` parser.
2. Recognize `.sql` files with a valid sqleton preamble.
3. Compile them through the shared compiler.
4. Add parser and repository tests.

### Phase 3: make aliases explicit at the same time

1. Introduce explicit alias detection for `.alias.yaml` and `.alias.yml`.
2. Stop using command-then-alias fallback for sqleton repositories.
3. Update repository examples to show `aliases/` directories.
4. Add tests proving alias and command files never cross-parse.

### Phase 4: decide migration strategy

Two clean options exist:

1. SQL-first migration:
   bulk-convert existing YAML command files and delete the YAML source format quickly.
2. Time-boxed transition:
   allow both YAML and SQL temporarily, but keep them compiling to one spec and set a removal date for YAML query bodies.

If you want the cleanest long-term design, prefer option 1. If the repo surface is too large to migrate in one pass, option 2 is acceptable as a temporary operational compromise.

### Phase 5: update scaffolding and docs

1. Make `select --create-query` emit `.sql`.
2. Update `sqleton/cmd/sqleton/doc/topics/06-query-commands.md`.
3. Add help examples showing the new format and explicit alias filenames.
4. Update README examples.

## API And File References

- Current YAML spec:
  `sqleton/pkg/cmds/sql.go`
- Current YAML loader:
  `sqleton/pkg/cmds/loaders.go`
- Current repository walking:
  `glazed/pkg/cmds/loaders/loaders.go`
- Current repository composition:
  `clay/pkg/repositories/repository.go`
- Current SQL runtime:
  `clay/pkg/sql/template.go`
  `clay/pkg/sql/query.go`
- `jsverbs` source-model inspiration:
  `go-go-goja/pkg/jsverbs/scan.go`
  `go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md`

## Open Questions

1. Should the preamble marker be `/* sqleton` exactly, or should it allow `/* sqleton-command` for future-proofing?
2. Do you want subquery fragments in v1, or is "use CTEs first" acceptable?
3. Should `select --create-query` immediately switch output format, or should it support a temporary `--format yaml|sql` flag during migration?
4. Is the long-term target SQL-only command sources, or dual-format support?
5. Do you want aliases to live in `aliases/` subdirectories by convention, or only rely on `.alias.yaml` naming?
6. Should legacy YAML command files remain plain `.yaml`, or should they get a temporary explicit suffix during migration?

## References

- `go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md`
- `go-go-goja/pkg/jsverbs/scan.go`
- `sqleton/pkg/cmds/loaders.go`
- `sqleton/pkg/cmds/sql.go`
- `sqleton/cmd/sqleton/cmds/select.go`
- `sqleton/cmd/sqleton/doc/topics/06-query-commands.md`
- `clay/pkg/sql/template.go`
- `clay/pkg/sql/query.go`
- `glazed/pkg/cmds/alias/alias.go`
