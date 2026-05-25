# Changes

## 0.1

- Converted initial direct CSV-to-GPX concept into staged CSV-to-SQLite import.
- Added `import` command for `gator` CSV files.
- Reserved `google` importer command but intentionally left it unimplemented.
- Added SQLite storage with de-duplication via stable point hash.
- Added `export` command to write stored columns as CSV.
- Added partial date filtering for export.
