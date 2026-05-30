# GoTravel Commands

This file defines the intended command-line interface. If code behaviour differs from this file, treat that as a bug unless `AUTHORITATIVE_SPECIFICATION.md` has deliberately changed.

## Database

```bash
GoTravel db init [--db gotravel.sqlite] [--force]
GoTravel db verify [--db gotravel.sqlite]
GoTravel db export [--db gotravel.sqlite] [--force] filename
GoTravel db import [--db gotravel.sqlite] [--force] filename
```

### Database Arguments

```text
filename   SQLite database file to export to or import from.
```

### Database Options

```text
--db PATH     SQLite database path. Defaults to gotravel.sqlite.
--force       Allow destructive or overwrite behaviour where applicable.
```

### Database Behaviour

`GoTravel db init` creates the SQLite database and required schema if missing. It is safe to run repeatedly. With `--force`, it replaces the existing database file before initialising a fresh schema.

`GoTravel db verify` validates that the configured database is a usable SQLite database and contains the required GoTravel tables.

`GoTravel db export filename` copies the whole configured SQLite database to `filename`. It refuses to overwrite an existing output file unless `--force` is supplied. It does not apply GPS date filters and does not transform rows.

`GoTravel db import filename` restores or copies a whole GoTravel SQLite database into the configured database path. It validates the input as a usable GoTravel database and refuses to overwrite an existing target unless `--force` is supplied. It does not merge or transform rows.

## Import

```bash
GoTravel import [--db gotravel.sqlite] [--force] gator input.csv
GoTravel import [--db gotravel.sqlite] [--force] gator -
```

### Import Arguments

```text
gator or google   Import format. gator is active; google is reserved.
input.csv         One or more CSV files to import.
-                 Read CSV from stdin.
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
- Do not skip bad data.

With `--force`:

- Skip corrupt rows.
- Store corrupt row details in `import_errors`.
- Commit valid rows.
- Report rows seen, imported, and skipped.

## Export

```bash
GoTravel export gator output.csv [--db gotravel.sqlite] [--force] [--start VALUE] [--stop VALUE]
GoTravel export gpx output.gpx [--db gotravel.sqlite] [--force] [--start VALUE] [--stop VALUE]
```

### Export Arguments

```text
gator or google or gpx   Export format. gator and gpx are active; google is reserved.
output.csv               Output CSV file path for gator/google export.
output.gpx               Output GPX file path for gpx export.
-                        Write to stdout.
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
YYYY-MM-DD HH
YYYY-MM-DD HH:MM
YYYY-MM-DD HH:MM:SS
```

### Gator Export Output Columns

Current staged CSV export columns:

```csv
dt,lat,lng,altitude,angle,speed,params
```

### GPX Export Behaviour

`GoTravel export gpx` writes a GPX 1.1 file containing one track with one segment, ordered by staged point timestamp.

It does not perform route matching, trip segmentation, dwell-time calculation, or provider calls.

## Route Matching

```bash
GoTravel route-match run [--db gotravel.sqlite] [--provider noop|ors|osrm|valhalla] [--profile VALUE] [--ors-base-url VALUE] [--osrm-base-url VALUE] [--from VALUE] [--to VALUE] [--radius METRES]
GoTravel route-match inspect [--db gotravel.sqlite] run-id
GoTravel route-match export [--db gotravel.sqlite] [--force] geojson run-id output.geojson
GoTravel route-match export [--db gotravel.sqlite] [--force] gpx run-id output.gpx
```

### Route Matching Arguments

```text
run-id          Stored route_match_runs ID.
output.geojson  GeoJSON output file path.
output.gpx      Matched-route GPX output file path.
-               Write export output to stdout.
```

### Route Matching Options

```text
--db PATH             SQLite database path. Defaults to gotravel.sqlite.
--provider NAME       Routing provider. Defaults to noop.
--profile VALUE       Routing profile. Defaults to driving.
--ors-base-url URL    Base URL for OpenRouteService when using the ors provider.
--osrm-base-url URL   Base URL for OSRM when using the osrm provider.
--from VALUE          Start date/time filter for source staged points.
--to VALUE            Stop date/time filter for source staged points.
--radius METRES       Optional route-match radius in metres.
--force               Allow overwriting existing route-match export files.
```

### Route Matching Behaviour

`route-match run` loads staged points, applies optional date filters, runs the provider-neutral route-match runner, persists the result, and prints a concise stored-run summary.

`route-match inspect` prints a stored-run summary plus linked point count and timestamps.

`route-match export geojson` writes a GeoJSON Feature from a stored route-match run. It passes stored GeoJSON through and converts supported encoded polyline geometry to GeoJSON LineString.

`route-match export gpx` writes matched route geometry as a GPX 1.1 track. It exports geometry only; it does not perform trip segmentation or attach original point timestamps to matched geometry.

Supported stored geometry formats for route-match export are GeoJSON, encoded polyline precision 5, and encoded polyline precision 6. Unsupported formats return clear errors.

## Safety Rules

- Never overwrite an existing output file unless `--force` is provided.
- Never apply overwrite checks when output is `-`.
- Never silently change command syntax.
- Database import/export commands must not transform staged rows.
- Route-match commands must keep provider-specific behaviour behind the routing provider layer.

## Reserved Future Commands

These are likely future commands but are not current required behaviour:

```bash
GoTravel export google output.csv
GoTravel export audit output.csv
GoTravel analyse routes
GoTravel report trips
GoTravel report map
```

Do not implement reserved commands unless explicitly requested.
