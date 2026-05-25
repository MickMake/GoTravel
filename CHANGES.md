# Changelog

## 0.4

- Added `GoTravel db init` to initialise the SQLite database and required schema.
- Added `GoTravel db export <filename>` to copy the whole SQLite database to a backup/transfer file.
- Added `GoTravel db import <filename>` to restore a whole GoTravel SQLite database file.
- Added database validation for import/export operations.
- Documented database commands in `COMMANDS.md` and `README.md`.

## 0.3

- Added `AUTHORITATIVE_SPECIFICATION.md` as the top-level behavioural specification.
- Added `COMMANDS.md` to lock down current CLI syntax and safety rules.
- Added `CODEX.md` to constrain future Codex work.
- Added `AGENTS.md` for general AI/code-agent workflow rules.
- Added updated `README.md` aligned with the current staged import/export scope.
- Added `TRACKER_SIGNALS.md.template` to avoid overwriting an existing tracker-signal specification.

## 0.2

- Split GoTravel into `cmd`, `import`, `export`, `routing`, `profiles`, `storage`, `examples`, and `tests`.
- Renamed the routing concept to `routing` instead of `ors`.
- Added canonical internal `Point` model.
- Added source metadata: `source_file`, `source_line`, and `imported_at`.
- Added `import_runs` table.
- Added `import_errors` table.
- Added `--force` support for imports: skip corrupt rows, store errors, and commit valid rows.
- Added default import safety: corrupt rows abort the file import unless `--force` is used.
- Added default export safety: existing output files are not overwritten unless `--force` is used.
- Preserved staged CSV export behaviour.
- Added basic tests and fixtures.

## 0.1

- Initial simple staged import/export implementation.
- Added Gator CSV import into SQLite.
- Added staged CSV export from SQLite.
- Added date/time filtering for export.
