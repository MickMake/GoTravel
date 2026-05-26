# GoTravel Authoritative Specification

## 1. Purpose

GoTravel is a deliberately simple command-line tool for importing GPS/tracker data from known CSV formats into SQLite, then exporting/reporting that staged data in controlled ways.

The project must remain boring, inspectable, and easy to run on private infrastructure. If a proposed feature requires a framework, daemon, queue, web UI, cloud account, or ceremonial chanting, it is probably out of scope until explicitly approved.

## 2. Current Scope

### Phase 1: Import and Export Staging

Required now:

- Initialise and verify the SQLite staging database.
- Import multiple CSV files.
- Support `gator` import first.
- Reserve `google` as a future importer/exporter.
- Support stdin import with `-`.
- Store normalised records in SQLite.
- Preserve import metadata.
- Detect and skip duplicate points.
- Abort on corrupt input unless `--force` is used.
- Export staged rows to Gator-style CSV/stdout with optional date filtering.
- Export staged rows to GPX 1.1/stdout with optional date filtering.
- Refuse to overwrite output files unless `--force` is used.

Not required yet:

- Google CSV import/export implementation.
- Audit export.
- KML export.
- ORS/OSRM/Valhalla route analysis.
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

## 6. Database Behaviour

Initialise the database:

```text
GoTravel db init [--db gotravel.sqlite] [--force]
```

- Create the SQLite database and required schema if missing.
- Be safe to run repeatedly.
- With `--force`, replace the existing database file before initialising a fresh schema.

Verify the database:

```text
GoTravel db verify [--db gotravel.sqlite]
```

- Validate that the configured file is a usable SQLite database.
- Validate that required GoTravel tables are present.

Whole-database copy/restore commands are supported but are not point import/export commands:

```text
GoTravel db export [--db gotravel.sqlite] [--force] <filename>
GoTravel db import [--db gotravel.sqlite] [--force] <filename>
```

- `db export` copies the whole SQLite database to another file.
- `db import` restores a whole SQLite database file.
- These commands must not transform rows or apply GPS date filters.

## 7. Import Behaviour

Default import behaviour:

```text
GoTravel import gator input.csv
GoTravel import gator input1.csv input2.csv
```

- Abort on the first corrupt row.
- Roll back the current file import.
- Report source file, source line, and reason.
- Do not partially commit that file.
- Skip duplicate points using the stable point hash.

Forced import behaviour:

```text
GoTravel import --force gator input.csv
```

- Skip corrupt rows.
- Record corrupt rows in `import_errors`.
- Commit valid rows.
- Skip duplicate points using the stable point hash.
- Print/import summary with seen/imported/skipped counts.

Stdin import:

```text
GoTravel import gator -
```

- Read CSV from stdin.
- Store `source_file` as `<stdin>`.
- Preserve actual source line numbers.

## 8. Export Behaviour

The export format must be explicit:

```text
GoTravel export <gator|google|gpx> <output.csv|output.gpx|-> [--db gotravel.sqlite] [--force] [--start VALUE] [--stop VALUE]
```

Gator CSV export:

```text
GoTravel export gator output.csv
GoTravel export gator -
```

- Export staged rows using the current Gator-style staged CSV columns.
- Refuse to overwrite an existing output file unless `--force` is supplied.
- Write to stdout when output is `-`.

GPX export:

```text
GoTravel export gpx output.gpx
GoTravel export gpx -
```

- Generate GPX 1.1.
- Emit one track containing one segment ordered by GPS timestamp.
- Preserve timestamp as GPX point time.
- Use latitude and longitude as GPX point attributes.
- Include elevation from the staged altitude value.
- Do not invent routes, stops, names, dwell times, or trip segmentation.
- Do not call routing providers.

Google export:

```text
GoTravel export google output.csv
```

- Reserved command shape only.
- Must return a clear not-implemented error until explicitly implemented.

## 9. Date Filter Behaviour

Export date filters must support partial precision:

```text
YYYY
YYYY-MM
YYYY-MM-DD
YYYY-MM-DD HH
YYYY-MM-DD HH:MM
YYYY-MM-DD HH:MM:SS
```

For `--start`, partial values resolve to the beginning of the specified period.

For `--stop`, partial values include the full specified period. For example:

```text
2025-05          includes all of May 2025
2025-05-02       includes that full day
2025-05-02 13    includes 13:00:00 through 13:59:59
2025-05-02 13:30 includes 13:30:00 through 13:30:59
```

## 10. Storage Rules

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

## 11. Routing Scope

The `routing/` package is reserved for future route processing.

Supported future providers:

- OpenRouteService
- OSRM
- Valhalla

The interface must remain provider-neutral. Do not hard-code the whole project around one routing provider.

The core routing interface must expose only operations shared by all supported providers. Provider-specific features must remain behind optional extensions, provider-specific packages, or raw provider responses until explicitly approved.

`ROUTING_PROVIDERS.md` defines the planned provider contract, core interface, common provider calls, and non-goals. Use it as the routing implementation reference before changing `routing/` code.

## 12. Presentation Scope

Current presentation/export formats:

- Gator-style staged CSV
- GPX 1.1

Future presentation formats may include:

- Audit CSV summaries
- KML
- HTML map reports

Do not implement these until explicitly requested and the staged import/export path remains solid and tested.

## 13. Change Control

Any implementation agent must:

- Read this file before changing code.
- Update `CHANGES.md` for behaviour changes.
- Update `COMMANDS.md` for CLI changes.
- Update tests for behavioural changes.
- Avoid speculative refactors.

This file is the top-level authority for GoTravel behaviour unless the user explicitly supersedes it.
