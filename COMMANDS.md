# GoTravel Commands

This file defines the intended command-line interface. If code behaviour differs from this file, treat that as a bug unless `AUTHORITATIVE_SPECIFICATION.md` has deliberately changed.

## Database

```bash
GoTravel db init [--db gotravel.sqlite] [--force]
GoTravel db verify [--db gotravel.sqlite]
GoTravel db export [--db gotravel.sqlite] [--force] <filename>
GoTravel db import [--db gotravel.sqlite] [--force] <filename>
```

### Database Arguments

```text
<filename>   SQLite database file to export to or import from.
```

### Database Options

```text
--db PATH     SQLite database path. Defaults to gotravel.sqlite.
--force       Allow destructive/overwrite behaviour where applicable.
```

### Database Behaviour

`GoTravel db init` creates the SQLite database and required schema if missing. It is safe to run repeatedly. With `--force`, it replaces the existing database file before initialising a fresh schema.

`GoTravel db verify` validates that the configured database is a usable SQLite database and contains the required GoTravel tables.

`GoTravel db export <filename>` copies the whole configured SQLite database to `<filename>`. It refuses to overwrite an existing output file unless `--force` is supplied. It does not apply GPS date filters and does not transform rows.

`GoTravel db import <filename>` restores/copies a whole GoTravel SQLite database into the configured database path. It validates the input as a usable GoTravel database and refuses to overwrite an existing target unless `--force` is supplied. It does not merge or transform rows.

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
- Database import/export commands must not transform staged rows.

## Reserved Future Commands

These are likely future commands but are not current required behaviour:

```bash
GoTravel export gator output.csv
GoTravel export google output.csv
GoTravel export audit output.csv
GoTravel export gpx output.gpx
GoTravel analyse routes
GoTravel report trips
GoTravel report map
```

Do not implement reserved commands unless explicitly requested.
