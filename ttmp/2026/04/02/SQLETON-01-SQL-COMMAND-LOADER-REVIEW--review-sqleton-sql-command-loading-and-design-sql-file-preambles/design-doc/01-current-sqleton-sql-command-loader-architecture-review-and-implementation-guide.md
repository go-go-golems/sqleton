---
Title: Current sqleton SQL command loader architecture review and implementation guide
Ticket: SQLETON-01-SQL-COMMAND-LOADER-REVIEW
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: clay/pkg/repositories/repository.go
      Note: Repository assembly
    - Path: clay/pkg/sql/query.go
      Note: Render and execute query helpers used by SqlCommand
    - Path: clay/pkg/sql/template.go
      Note: Template helper surface used by sqleton SQL commands
    - Path: glazed/pkg/cmds/alias/alias.go
      Note: Alias model and runtime forwarding behavior used to assess how aliases complicate the loader design
    - Path: glazed/pkg/cmds/loaders/loaders.go
      Note: Generic file discovery
    - Path: sqleton/cmd/sqleton/main.go
      Note: Application composition
    - Path: sqleton/pkg/cmds/loaders.go
      Note: Sqleton-specific YAML loader and direct decode-to-runtime-command path
    - Path: sqleton/pkg/cmds/sql.go
      Note: SqlCommand structure
ExternalSources: []
Summary: Evidence-based review of how sqleton loads YAML-backed SQL commands today, where the design is elegant, where it is awkward, and how to refactor it around a common parsed spec.
LastUpdated: 2026-04-02T08:50:58.081401955-04:00
WhatFor: Explain how sqleton loads YAML-backed SQL commands today, assess the architecture, and propose a cleaner internal shape for future loader work.
WhenToUse: Read this before changing sqleton command loading, repository registration, YAML command specs, or the run-command execution path.
---



# Current sqleton SQL command loader architecture review and implementation guide

## Executive Summary

`sqleton` already has a useful core idea: SQL commands are mostly declarative, live in the filesystem, inherit their command path from directories, and compile into normal Glazed/Cobra commands. That part is elegant.

The design becomes much less clean at the loading boundary. Today the system has no single "parsed command spec" phase. Instead, the YAML loader decodes directly into a runtime `SqlCommand`, the command constructor injects shared parameter layers, the repository walker decides parent paths from directories, Glazed loader helpers silently blur command-vs-alias parsing, and `main.go` adds a one-off fast path for `run-command` outside the usual Cobra registration flow. The result works, but it is harder than it should be to reason about, extend, or diagnose.

The most important recommendation in this review is structural:

- separate source parsing from command compilation,
- create one internal `SqlCommandSpec`,
- make repository walking care only about source discovery and hierarchy,
- make command builders care only about turning validated specs into runtime commands,
- and collapse ad-hoc entrypoints like `run-command` back into the same registration model.

That refactor would materially improve the current YAML path and also make the future SQL-file format straightforward to add.

## Problem Statement

The user request was not only "explain how this works", but also "assess / review the design for elegance, clarity, and suggest improvements". The key problem therefore is architectural clarity.

An intern reading this subsystem today has to hold too many concepts in their head at once:

- `sqleton` defines the SQL command schema and execution behavior.
- `glazed` defines generic filesystem command loaders, alias fallback, and parent-path derivation.
- `clay` defines repositories, help loading, and command-tree registration.
- `sqleton/main.go` mixes static commands, repository loading, embedded query registration, and a special pre-Cobra execution path.

There is no single file or type that answers the simple question: "What is the neutral, validated representation of one SQL command before it becomes a Cobra command?" That absence is what makes the loader feel weird.

Scope of this document:

- in scope:
  current YAML SQL command files, the repository load path, Cobra command construction, `run-command`, `select --create-query`, and the SQL execution boundary in `clay/pkg/sql`
- out of scope:
  a full redesign of SQL templating semantics, database connection UX, or Glazed broadly outside the pieces that shape sqleton loading

## Proposed Solution

The proposed solution is an internal cleanup, not an end-user feature by itself.

Introduce a staged pipeline with explicit boundaries:

```text
filesystem entry
    |
    v
source parser
  - yaml parser
  - future sql-file parser
    |
    v
SqlCommandSpec
  - neutral validated representation
    |
    v
command compiler
  - apply shared layers
  - build SqlCommand
    |
    v
repository registration
  - parents/source/help tree
    |
    v
cli registration
  - BuildCobraCommand(...)
```

The important change is that parsing should stop before runtime policy begins. Today those are mixed.

Suggested internal types:

```go
type SqlCommandSpec struct {
    Name       string
    Short      string
    Long       string
    Layout     []*layout.Section
    Flags      []*parameters.ParameterDefinition
    Arguments  []*parameters.ParameterDefinition
    Layers     []layers.ParameterLayer
    Tags       []string
    Metadata   map[string]interface{}
    Query      string
    SubQueries map[string]string
    Format     string
}

type SqlCommandCompiler struct {
    DBConnectionFactory clay_sql.DBConnectionFactory
    ExtraLayers         []layers.ParameterLayer
}
```

And the compiler boundary:

```go
func ParseYAMLSqlCommandSpec(r io.Reader) (*SqlCommandSpec, error)
func ValidateSqlCommandSpec(spec *SqlCommandSpec) error
func (c *SqlCommandCompiler) Compile(spec *SqlCommandSpec, options ...cmds.CommandDescriptionOption) (*SqlCommand, error)
```

That same `SqlCommandSpec` would later be produced by a `.sql` parser, which is why the cleanup should happen before or together with the SQL-file work.

## System At A Glance

The current runtime path is easiest to understand as two parallel stories.

Normal repository-backed commands:

```text
queries/*.yaml
  -> sqleton/pkg/cmds/SqlCommandLoader.IsFileSupported
  -> glazed/pkg/cmds/loaders.LoadCommandsFromFS
  -> sqleton/pkg/cmds/loadSqlCommandFromReader
  -> sqleton/pkg/cmds.NewSqlCommand
  -> clay/pkg/repositories.Repository.Add
  -> clay/pkg/repositories.LoadRepositories
  -> sql.BuildCobraCommandWithSqletonMiddlewares / cli.BuildCobraCommand
  -> sqleton <parents> <name>
```

Ad-hoc file execution:

```text
sqleton run-command path/to/file.yaml
  -> main.go pre-Cobra branch
  -> FileNameToFsFilePath
  -> SqlCommandLoader.LoadCommands
  -> BuildCobraCommandWithSqletonMiddlewares
  -> mutate os.Args
  -> rootCmd.Execute()
```

That second path is the strongest evidence that the current loader boundaries are wrong: the application has to step outside normal command registration to execute a command file dynamically.

## Current Architecture

### 1. Query files are stored in command-shaped directories

The repository convention is simple and good. Query files live under a query root, and their directories become parent commands. `sqleton` documents this directly in `sqleton/cmd/sqleton/doc/topics/06-query-commands.md:46-80`.

Examples in the embedded tree include:

- `sqleton/cmd/sqleton/queries/mysql/ps.yaml`
- `sqleton/cmd/sqleton/queries/wp/ls-posts.yaml`
- `sqleton/cmd/sqleton/queries/pg/connections.yaml`

The pleasant property here is that authors do not manually restate command parents in every file. The filesystem is the source of truth for hierarchy.

### 2. `main.go` composes built-in commands and repository-backed commands together

`sqleton/cmd/sqleton/main.go:166-339` does four jobs in one function:

- register fixed commands like `db`, `run`, `select`, `query`, `serve`
- assemble repository search paths from config plus `$HOME/.sqleton/queries`
- construct one `SqlCommandLoader`
- load repository commands and build Cobra commands for them

This is workable, but it means loading policy, application composition, and command definition policy are all colocated.

The embedded query repository is mounted with:

- `queriesFS` from `sqleton/cmd/sqleton/main.go:142-143`
- `repositories.Directory{FS: queriesFS, RootDirectory: "queries", RootDocDirectory: "queries/doc", ...}` at `sqleton/cmd/sqleton/main.go:235-242`

That embedded/static-plus-runtime directory composition is one of the better parts of the design.

### 3. File discovery is generic Glazed loader infrastructure

`clay/pkg/repositories/repository.go:104-223` delegates filesystem walking to `glazed/pkg/cmds/loaders.LoadCommandsFromFS`, while also applying:

- `cmds.WithStripParentsPrefix([]string{directory.RootDirectory})`
- `alias.WithStripParentsPrefix([]string{directory.RootDirectory})`

`glazed/pkg/cmds/loaders/loaders.go:104-181` then:

- recursively walks directories
- calls `loader.IsFileSupported(...)`
- derives parents from the current directory with `GetParentsFromDir`
- injects `WithSource` and `WithParents`
- continues scanning even when individual files fail to load

That gives the loader a lot of implicit context, but it also means a reader has to know both the repository code and the Glazed helper to understand final command paths.

### 4. `SqlCommandLoader` only understands YAML today

The sqleton-specific loader lives in `sqleton/pkg/cmds/loaders.go:17-99`.

Its contract is narrow:

- `IsFileSupported(...)` delegates to `loaders.CheckYamlFileType(..., "sqleton")`
- `LoadCommands(...)` opens a file and calls `LoadCommandOrAliasFromReader(...)`
- `loadSqlCommandFromReader(...)` decodes YAML into `SqlCommandDescription`
- it then constructs a `SqlCommand` directly

This direct decode-to-runtime-command step is the main conceptual weakness. There is no intermediate "spec" or "AST" for command definitions.

### 5. The loader helper conflates command parsing and alias parsing

`glazed/pkg/cmds/loaders/loaders.go:38-65` reads the entire file into memory and:

1. tries the provided command parser,
2. if that fails for any reason, retries as an alias YAML file.

That is convenient, but it harms error clarity:

- an actually broken command file is not clearly distinguished from "this was an alias"
- the sqleton loader does not choose that behavior; it inherits it
- command parsing and alias parsing are treated as sibling fallbacks rather than explicit file kinds

For a system where design clarity matters, this is a weak seam.

### 6. Type detection is too permissive

`glazed/pkg/cmds/loaders/loaders.go:68-93` returns `true` when `type == "sqleton"` or when `type == ""`.

That means any YAML file without an explicit `type` field looks like a sqleton command candidate.

Inside a dedicated `queries/` tree that may be acceptable. Architecturally, though, it means:

- the loader is not strongly format-discriminating,
- adding more file kinds in the same tree becomes ambiguous,
- and a missing `type` is treated as intentional instead of as an explicit choice.

That is part of why the loader feels fuzzy rather than crisp.

### 7. `NewSqlCommand` mixes description construction and runtime policy

`sqleton/pkg/cmds/sql.go:118-155` creates a command and automatically appends shared layers:

- `sql-helpers`
- `sql-connection`
- `dbt`
- `glazed`

This is an important runtime policy choice, but it happens inside the command constructor itself.

That means the same function is responsible for:

- command identity,
- query storage,
- shared layer injection,
- and therefore part of CLI behavior

For a mature system, it would be cleaner if parsing produced a pure spec and a separate compiler or builder applied standard shared layers.

### 8. Execution lives in `SqlCommand`, but it also relies on hidden mutable state

`sqleton/pkg/cmds/sql.go:209-286` renders the query into `s.renderedQuery`, then later `RunQueryIntoGlaze(...)` uses that field. The code comment at `sqleton/pkg/cmds/sql.go:279-282` explicitly notes the cleanup debt.

This is not catastrophic, but it is another sign that stages are blurred:

- query rendering,
- execution,
- and command state

would be easier to test and reason about if they flowed through function arguments instead of mutable receiver state.

### 9. The SQL templating/runtime layer is actually fairly clean

`clay/pkg/sql/template.go` and `clay/pkg/sql/query.go` are not the weird part of the system.

They define:

- template helper functions such as `sqlStringIn`, `sqlDate`, `sqlColumn`, `sqlSingle`
- the render path via `RenderQuery(...)`
- the execution path via `RunQueryIntoGlaze(...)`

That separation is serviceable and consistent with sqleton's purpose. The main issue is above this layer, where source parsing and command registration are composed.

### 10. `run-command` is effectively implemented twice

`sqleton/cmd/sqleton/main.go:77-128` contains a special-case branch that intercepts `sqleton run-command file.yaml` before normal Cobra setup, loads the file, builds a command, mutates `os.Args`, and then falls back into `rootCmd.Execute()`.

At the same time, `sqleton/cmd/sqleton/main.go:130-137` defines a `run-command` Cobra command whose `Run` just panics with `not implemented`.

This is a concrete design smell:

- the public CLI surface says there is a normal command,
- the real implementation bypasses the normal command path,
- and the stub exists only to satisfy help/registration shape

That is the sort of thing that confuses maintainers quickly.

### 11. `README` promises a remote-file capability that the implementation does not support

The README example at `sqleton/README.md:304-314` shows:

```bash
sqleton run-command https://github.com/myorg/sql-commands/user-stats.yaml
```

But `main.go` passes the string into `loaders.FileNameToFsFilePath(...)`, whose implementation in `glazed/pkg/cmds/loaders/loaders.go:204-241` only supports local filesystem path shapes. No HTTP fetch path exists there.

That is not merely a doc bug. It is evidence that the current mental model of "loader input" is not clearly bounded.

### 12. `select --create-query` shows the intended declarative shape, but also shows the YAML lock-in

`sqleton/cmd/sqleton/cmds/select.go:106-190` can scaffold a new query command by creating a `SqlCommand` in memory and marshaling it back to YAML.

That is useful because it proves the declarative command idea is important to the product.

It also shows the cost of the current format choice:

- the command authoring story is assumed to end in YAML,
- the generated query body is embedded into YAML text,
- and that choice leaks into tooling and UX

This matters because the proposed SQL-file format should preserve the good part of this feature, which is command scaffolding, while changing only the storage format.

## Design Review

### What is elegant today

- Directory layout maps cleanly to command hierarchy.
- Embedded and user-local repositories can be merged in one tree.
- SQL rendering and SQL execution are mostly separate concerns.
- Declarative command definitions integrate cleanly with Glazed/Cobra once loaded.
- `select --create-query` demonstrates that "commands as source files" is a productive model.

### What is unclear or inelegant

1. There is no neutral parsed representation between file format and runtime command.
2. The loader pipeline is spread across four packages, so the conceptual center is missing.
3. Generic helper fallback behavior makes error reporting and file-kind reasoning weaker.
4. `run-command` is implemented by escaping the normal registration model.
5. Type detection and source-kind detection are permissive rather than explicit.
6. Some SQL examples still interpolate strings unsafely, for example `sqleton/cmd/sqleton/queries/pg/connections.yaml:49-63`, which undermines the "safe templating" story documented in `clay/pkg/doc/topics/03-sql-commands.md:203-348`.

## Design Decisions

### Decision 1: Introduce one internal `SqlCommandSpec`

Rationale:

- it creates a stable semantic center,
- it allows multiple source formats without multiple runtime models,
- and it makes validation explicit before command construction.

### Decision 2: Move shared-layer injection into a compiler/builder stage

Rationale:

- parsing should not decide runtime policy,
- builder configuration should be testable in isolation,
- and future formats should not need to know about all shared layers.

### Decision 3: Make command-vs-alias dispatch explicit

Rationale:

- sqleton query files should parse as sqleton query files,
- alias fallback should not hide format errors,
- and format discrimination should become simpler if `.sql` sources are added.

What "explicit" means in practice:

- command source files should be identified deterministically before full parsing
- alias source files should be identified deterministically before full parsing
- the sqleton loader should never infer "this must have been an alias" only because command parsing failed

Recommended differentiation scheme:

- commands:
  `.sql` for the new SQL-first format
- aliases:
  `.alias.yaml` or `.alias.yml`
- optional layout convention:
  place alias files under an `aliases/` subdirectory so the tree is self-explanatory to humans

Recommended repository example:

```text
queries/
  wp/
    ls-posts.sql
    posts-counts.sql
    aliases/
      drafts.alias.yaml
      newest.alias.yaml
```

This choice matters because it shrinks the loader problem substantially. Once file kind is known up front, command parsing errors stay command parsing errors, alias parsing errors stay alias parsing errors, and the two concerns stop contaminating each other.

### Decision 4: Collapse `run-command` into the normal command system

Rationale:

- one CLI registration path is easier to explain,
- help behavior becomes honest,
- and future formats benefit automatically.

### Decision 5: Treat source format as a parser concern, not an execution concern

Rationale:

- YAML and SQL should both compile to the same spec,
- execution should remain `RenderQuery(...)` plus `RunQueryIntoGlaze(...)`,
- and repository walking should not care what format the source originally used.

## Alternatives Considered

### Keep the current system and only bolt on a `.sql` parser

This is the lowest-effort path, but it would preserve the core problem. A `.sql` parser added directly into the current loader would create two parallel "decode directly into runtime command" paths and make the weirdness worse rather than better.

### Replace filesystem repositories with a custom sqleton-only loader stack

This would simplify local reasoning inside sqleton, but it would throw away useful `clay` and `glazed` abstractions:

- directory-to-command hierarchy
- help loading
- repository merging
- general command-tree registration

That is too much churn for the problem at hand.

### Keep YAML as the permanent only source format

This avoids refactoring but preserves the main authoring pain point: SQL lives inside YAML block strings. That makes editor support, diff readability, and large-query authoring worse than necessary.

## Recommended Refactor Shape

### Stage A: create a spec package boundary

Pseudocode:

```go
type SqlCommandSpecParser interface {
    Supports(path string, contents []byte) bool
    Parse(path string, contents []byte) (*SqlCommandSpec, error)
}

type SqlCommandRegistryLoader struct {
    Parsers  []SqlCommandSpecParser
    Compiler *SqlCommandCompiler
}
```

Responsibilities:

- parser:
  source-format concerns only
- compiler:
  shared layers, `SqlCommand` creation, validation of runtime contract
- repository:
  parents, source names, help pages, mounted directories
- CLI:
  Cobra wiring only

### Stage A.1: make aliases a separate source kind

Add one explicit dispatch layer ahead of parsing:

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

Recommended rules:

- `*.sql` with a valid sqleton preamble:
  `SourceSQLCommand`
- `*.alias.yaml` or `*.alias.yml`:
  `SourceYAMLAlias`
- transition-only support for existing YAML commands:
  `SourceYAMLCommand`
- anything else:
  `SourceUnknown`

That gives sqleton one deterministic dispatcher instead of one parser with an alias fallback side channel.

### Stage B: make repository errors explicit

For sqleton command repositories, malformed files should fail loudly in development and CI. Silent drop-on-warning behavior is dangerous for command catalogs.

A practical approach is:

- strict mode in tests and embedded repository load
- optional soft mode only for hot-reload/watch flows if needed later

### Stage C: unify dynamic-file execution with repository execution

Instead of:

- special pre-Cobra load,
- synthetic `os.Args` rewriting,
- stub command panic

use:

```go
func NewRunCommandFileCommand(loader *SqlCommandRegistryLoader) cmds.Command
```

which:

- parses one file into `SqlCommandSpec`
- compiles it
- executes it through the same Cobra builder path as any other command

## Implementation Plan

### Phase 1: internal cleanup with no source-format change

1. Extract `SqlCommandSpec` and `ParseYAMLSqlCommandSpec(...)`.
2. Change `SqlCommandLoader` to parse to spec first and compile second.
3. Add validation tests for spec parsing and compilation separately.
4. Remove the `run-command` panic/stub split and implement one honest path.

### Phase 2: make aliases explicit and harden loader boundaries

1. Introduce `DetectSourceKind(path, contents)` for sqleton-owned source dispatch.
2. Define alias files explicitly as `.alias.yaml` or `.alias.yml`.
3. Optionally constrain aliases further to `aliases/` subdirectories for readability.
4. Stop using `LoadCommandOrAliasFromReader(...)` for sqleton command repositories.
5. Route:
   - SQL command files to `ParseSQLFileSpec(...)`
   - YAML command files to `ParseYAMLSqlCommandSpec(...)`
   - alias files to `alias.NewCommandAliasFromYAML(...)`
6. Tighten repository load behavior so malformed embedded queries fail tests and startup.

### Phase 3: add `.sql` source parsing

1. Implement `ParseSQLFileSqlCommandSpec(...)`.
2. Compile it through the same compiler from Phase 1.
3. Decide migration policy:
   either bulk-migrate and drop YAML quickly, or keep a short, explicit transition window.

### Phase 4: simplify runtime entrypoints and scaffolding

1. Remove the `run-command` panic/stub split and implement one honest path.
2. Make `select --create-query` emit `.sql` command files.
3. Update help pages and examples to show explicit alias naming and SQL-first command sources.

## Testing And Validation Strategy

Add tests at each stage boundary.

Parser tests:

- valid YAML command
- missing `short`
- missing `query`
- bad parameter definitions
- ambiguous file type
- `.alias.yaml` dispatches to alias parsing and never to command parsing
- malformed `.sql` command never falls through to alias parsing
- malformed `.alias.yaml` never reports a command-parse error first

Compiler tests:

- layers are injected exactly once
- source and parent options are preserved
- `SqlCommand.IsValid()` is enforced through spec validation rather than only late object checks

Repository tests:

- nested directories create expected `FullPath()` values
- malformed files fail with actionable errors
- embedded and local repositories merge without path collisions

CLI tests:

- `sqleton help`
- repository command registration
- `run-command` file execution
- `select --create-query` roundtrip against the current source format
- explicit alias registration and resolution from `aliases/*.alias.yaml`

Manual checks:

```bash
go test ./sqleton/pkg/cmds ./clay/pkg/repositories ./glazed/pkg/cmds/loaders
go run ./sqleton/cmd/sqleton queries --fields name,source
go run ./sqleton/cmd/sqleton mysql ps --print-query
go run ./sqleton/cmd/sqleton run-command ./sqleton/cmd/sqleton/queries/mysql/ps.yaml --print-query
```

## Open Questions

1. Do you want a short migration window with YAML and SQL side by side, or a direct move to SQL-first sources?
2. Do you want aliases to stay in the same repository tree under `aliases/`, or move to a separate mounted alias repository entirely?
3. Should load failures during startup be fatal by default for embedded queries?
4. Is remote `run-command` actually desired, or should the README example be removed instead of implemented?
5. Do you want a temporary explicit YAML command suffix as well, such as `.cmd.yaml`, or is plain legacy `.yaml` sufficient during migration?

## References

- `sqleton/cmd/sqleton/main.go`
- `sqleton/pkg/cmds/loaders.go`
- `sqleton/pkg/cmds/sql.go`
- `sqleton/pkg/codegen/codegen.go`
- `sqleton/cmd/sqleton/cmds/select.go`
- `glazed/pkg/cmds/loaders/loaders.go`
- `glazed/pkg/doc/topics/22-command-loaders.md`
- `clay/pkg/repositories/repository.go`
- `clay/pkg/sql/template.go`
- `clay/pkg/sql/query.go`
- `clay/pkg/doc/topics/03-sql-commands.md`
- `sqleton/cmd/sqleton/doc/topics/06-query-commands.md`
- `glazed/pkg/cmds/alias/alias.go`
