---
Title: Add DuckDB support to sqleton
Ticket: SQLETON-03-DUCKDB-SUPPORT
Status: active
Topics:
    - backend
    - duckdb
    - database
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: clay/pkg/sql/config.go
      Note: Core database config with Connect()
    - Path: clay/pkg/sql/flags/sql-connection.yaml
      Note: Connection flag YAML definition
    - Path: clay/pkg/sql/query.go
      Note: RunQueryIntoGlaze() - query execution pipeline
    - Path: clay/pkg/sql/settings.go
      Note: DBConnectionFactory type and parameter layers
    - Path: clay/pkg/sql/sources.go
      Note: Source.ToConnectionString() - needs DuckDB case
    - Path: sqleton/cmd/sqleton/cmds/db.go
      Note: DB subcommand and driver imports
    - Path: sqleton/cmd/sqleton/main.go
      Note: CLI entrypoint and command initialization
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-04T21:01:08.234237965-04:00
WhatFor: ""
WhenToUse: ""
---


# Add DuckDB support to sqleton

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- backend
- duckdb
- database

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
