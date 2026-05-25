# GoTravel

GoTravel is a deliberately simple command-line tool for importing tracker/GPS CSV data into SQLite, then exporting or analysing that staged data.

The project favours boring, inspectable workflows over clever machinery. It is intended to run locally or on private infrastructure without requiring cloud services.

## Current Status

Current focus:

- Initialise, verify, export, and import the SQLite staging database.
- Import Gator CSV data.
- Stage normalised points in SQLite.
- Preserve source metadata.
- Record corrupt rows when forced.
- Export staged rows to CSV/stdout.
- Export staged rows to GPX.

Reserved for later:

- Google import.
- Audit export.
- KML export.
- OpenRouteService/OSRM route analysis.
- Reports and maps.

## Commands

Initialise the database:

```bash
GoTravel db init
GoTravel db init --db gotravel.sqlite
```

Verify the database:

```bash
GoTravel db verify
GoTravel db verify --db gotravel.sqlite
```

Export a whole database copy:

```bash
GoTravel db export backup.sqlite
GoTravel db export --force backup.sqlite
```

Import/restore a whole database copy:

```bash
GoTravel db import backup.sqlite
GoTravel db import --force backup.sqlite
```

Import one or more Gator CSV files:

```bash
GoTravel import gator input.csv
GoTravel import gator input1.csv input2.csv
```

Import from stdin:

```bash
cat input.csv | GoTravel import gator -
```

Use a specific database:

```bash
GoTravel import --db gotravel.sqlite gator input.csv
```

Continue past corrupt rows:

```bash
GoTravel import --force gator input.csv
```

Export staged rows to Gator-style CSV:

```bash
GoTravel export gator output.csv
```

Export to stdout:

```bash
GoTravel export gator -
```

Export with a date range:

```bash
GoTravel export gator output.csv --start 2025-05 --stop 2025-06
GoTravel export gator output.csv --start "2025-05-02 13" --stop "2025-05-02 13:30"
```

Export staged rows to GPX:

```bash
GoTravel export gpx output.gpx
GoTravel export gpx output.gpx --start 2025-05 --stop 2025-06
```

Overwrite an existing output file:

```bash
GoTravel export gator --force output.csv
GoTravel export gpx --force output.gpx
```

## Current Gator Export Columns

```csv
dt,lat,lng,altitude,angle,speed,params
```

## GPX Export

`GoTravel export gpx` writes GPX 1.1 with one track and one segment from staged points ordered by timestamp.

It does not perform route matching, trip segmentation, dwell-time calculation, or provider calls.

## Date Filters

Supported date/time filters:

```text
YYYY
YYYY-MM
YYYY-MM-DD
YYYY-MM-DD HH
YYYY-MM-DD HH:MM
YYYY-MM-DD HH:MM:SS
```

For `--stop`, partial values include the full specified period. For example, `--stop "2025-05-02 13"` includes that full hour.

## Repository Layout

```text
cmd/        CLI command handling
examples/   generic examples
export/     exporting code
import/     importing code
profiles/   import/export profile files
routing/    future OpenRouteService/OSRM support
storage/    SQLite and file handling
tests/      tests and fixtures
```

## Build

```bash
go mod tidy
go build -o GoTravel .
```

## Test

```bash
go test ./...
```

## Agent/Codex Notes

Before using Codex or another code agent, read:

```text
AUTHORITATIVE_SPECIFICATION.md
COMMANDS.md
CODEX.md
AGENTS.md
```

These files exist to stop GoTravel turning into a cathedral with a GPS antenna.
