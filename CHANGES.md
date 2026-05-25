# Changelog

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
