# Changelog

## 2026-04-02

- Initial workspace created
- Added a detailed design and implementation guide for removing Viper from sqleton and separating app config from command section config
- Recorded the comparison with pinocchio as the main architectural reference
- Uploaded the ticket bundle to reMarkable as `SQLETON-02 Viper and App Config Cleanup` in `/ai/2026/04/02/SQLETON-02-VIPER-APP-CONFIG-CLEANUP`
- Implemented an app-owned `sqleton` config loader with direct YAML decoding and `SQLETON_REPOSITORIES` environment merging
- Added focused tests for empty config, YAML config loading, environment repository parsing, and config-plus-environment merging
- Verified the existing SQLite and configured-repository smoke tests still pass before any startup migration
- Replaced `clay.InitViper("sqleton", ...)` with `clay.InitGlazed("sqleton", ...)`
- Removed direct `viper.GetStringSlice("repositories")` usage from `sqleton` startup and switched repository discovery to the app-owned config loader
- Verified the `sqleton/...` test tree still passes after the Viper removal
- Added an sqleton-owned parser config helper so command config files are only loaded from explicit `--config-file`
- Added smoke coverage for repository discovery from `~/.sqleton/config.yaml` and for explicit `--config-file` command config loading
- Removed the remaining direct `viper` reads from `sqleton/cmd/sqleton/cmds/db.go`
- Verified there are no direct `viper` references left in `sqleton/cmd/sqleton` or `sqleton/pkg`
- Added a cross-ticket full-day project report to the ticket docs, mirroring the Obsidian project note and summarizing the entire 2026-04-02 sqleton cleanup from SQL loader redesign through Viper removal
