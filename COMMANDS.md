# GoTravel Commands

This file defines the intended command-line interface. If code behaviour differs from this file, treat that as a bug unless `AUTHORITATIVE_SPECIFICATION.md` has deliberately changed.

## Import

```bash
GoTravel import [--db gotravel.sqlite] [--force] <gator|google> <input.csv> [...]
GoTravel import [--db gotravel.sqlite] [--force] <gator|google> -
```

### Import Arguments

```text
<gator|google>   Import format. gator is active; google is reserved.
<input.csv>      One or more CSV files to import.
-                Read CSV from stdin.
```

### Import Options

```text
--db PATH     SQLite database path. Defaults to gotravel.sqlite.
--force       Continue past corrupt rows, record errors, and commit valid rows.
```

### Import Default Behaviour

Without `--force`:

- Abort on first corrupt row.
- Roll back the current file import.
- Report file, line, and error.
- Do not silently skip bad data.

With `--force`:

- Skip corrupt rows.
- Store corrupt row details in `import_errors`.
- Commit valid rows.
- Report rows seen/imported/skipped.

## Export

```bash
GoTravel export [--db gotravel.sqlite] [--force] <output.csv|-> [--start VALUE] [--stop VALUE]
```

### Export Arguments

```text
<output.csv>   Output CSV file path.
-              Write to stdout.
```

### Export Options

```text
--db PATH       SQLite database path. Defaults to gotravel.sqlite.
--force         Allow overwriting existing output files.
--start VALUE   Start date/time filter.
--stop VALUE    Stop date/time filter.
```

### Export Date Formats

Supported partial date/time formats:

```text
YYYY
YYYY-MM
YYYY-MM-DD
YYYY-MM-DD HH:MM:SS
```

### Export Output Columns

Current staged CSV export columns:

```csv
dt,lat,lng,altitude,angle,speed,params
```

## Safety Rules

- Never overwrite an existing output file unless `--force` is provided.
- Never apply overwrite checks when output is `-`.
- Never silently ignore corrupt input.
- Never silently change command syntax.

## Reserved Future Commands

These are likely future commands but are not current required behaviour:

```bash
GoTravel export gpx output.gpx
GoTravel analyse routes
GoTravel report trips
GoTravel report map
```

Do not implement reserved commands unless explicitly requested.
