# GoTravel Authoritative Specification

## 1. Purpose

GoTravel is a deliberately simple command-line tool for importing GPS/tracker data from known CSV formats into SQLite, then exporting/reporting that staged data in controlled ways.

The project must remain boring, inspectable, and easy to run on private infrastructure. If a proposed feature requires a framework, daemon, queue, web UI, cloud account, or ceremonial chanting, it is probably out of scope until explicitly approved.

## 2. Current Scope

### Phase 1: Import and Export Staging

Required now:

- Import multiple CSV files.
- Support `gator` first.
- Reserve `google` as a future importer.
- Support stdin import with `-`.
- Store normalised records in SQLite.
- Preserve import metadata.
- Detect duplicates.
- Abort on corrupt input unless `--force` is used.
- Export staged rows to CSV/stdout with optional date filtering.
- Refuse to overwrite output files unless `--force` is used.

Not required yet:

- GPX export.
- KML export.
- ORS/OSRM route analysis.
- HTML reports.
- Web UI.
- Background services.

## 3. Guiding Rules

1. Keep the tool simple.
2. Prefer explicit formats over auto-magic guessing.
3. Prefer SQLite over server databases for local staging.
4. Prefer files/stdin/stdout over web workflows.
5. Never silently discard data.
6. Never silently overwrite files.
7. Never introduce a large dependency without a clear reason.
8. Behaviour changes must be reflected in tests and documentation.

## 4. Repository Layout

```text
cmd/        CLI command handling only
examples/   small generic examples and fixtures
export/     export implementations
import/     import implementations
profiles/   import/export mapping profiles
routing/    route provider interfaces and implementations
storage/    SQLite, files, schema, point model, persistence
tests/      integration-style tests and fixtures
```

## 5. Canonical Internal Point Model

The internal staged record must represent both the GPS point and its import provenance.

Required fields:

```text
dt           GPS point timestamp
lat          latitude
lng          longitude
altitude     altitude, if present
angle        heading/angle, if present
speed        speed, if present
params       raw/normalised tracker parameters
format       import format, e.g. gator
source_file  input filename or <stdin>
source_line  original input line number
imported_at  time GoTravel imported the row
point_hash   duplicate-detection hash
```

## 6. Import Behaviour

Default import behaviour:

```text
GoTravel import gator input.csv
```

- Abort on the first corrupt row.
- Roll back the current file import.
- Report source file, source line, and reason.
- Do not partially commit that file.

Forced import behaviour:

```text
GoTravel import --force gator input.csv
```

- Skip corrupt rows.
- Record corrupt rows in `import_errors`.
- Commit valid rows.
- Print/import summary with seen/imported/skipped counts.

Stdin import:

```text
GoTravel import gator -
```

- Read CSV from stdin.
- Store `source_file` as `<stdin>`.
- Preserve actual source line numbers.

## 7. Export Behaviour

Default export behaviour:

```text
GoTravel export output.csv
```

- Refuse to overwrite an existing output file.
- Support date filtering with `--start` and `--stop`.

Forced export behaviour:

```text
GoTravel export --force output.csv
```

- Allow overwriting the output file.

Stdout export:

```text
GoTravel export -
```

- Write to stdout.
- Never apply overwrite checks to stdout.

## 8. Date Filter Behaviour

Export date filters must support partial precision:

```text
YYYY
YYYY-MM
YYYY-MM-DD
YYYY-MM-DD HH:MM:SS
```

The CLI may accept additional compatible timestamp formats later, but the above are the minimum required forms.

## 9. Storage Rules

SQLite is the staging store.

Required tables:

```text
points
import_runs
import_errors
```

Duplicate detection must use a stable hash of point content only:

```text
dt + lat + lng + altitude + angle + speed + params
```

Do not include provenance fields such as `source_file`, `source_line`, or `imported_at` in the hash.

## 10. Routing Scope

The `routing/` package is reserved for future route processing.

Supported future providers:

- OpenRouteService
- OSRM

The interface must remain provider-neutral. Do not hard-code the whole project around one routing provider.

## 11. Presentation Scope

Future presentation formats may include:

- CSV summaries
- GPX
- KML
- HTML map reports

Do not implement these until the staged import/export path is solid and tested.

## 12. Change Control

Any implementation agent must:

- Read this file before changing code.
- Update `CHANGES.md` for behaviour changes.
- Update `COMMANDS.md` for CLI changes.
- Update tests for behavioural changes.
- Avoid speculative refactors.

This file is the top-level authority for GoTravel behaviour unless the user explicitly supersedes it.
