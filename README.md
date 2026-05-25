# GoTravel

GoTravel is a deliberately small GPS import/export tool.

Version 0.2 focuses on staging Gator CSV files into SQLite and exporting staged rows back to CSV. GPX, routing, reports, and map views are reserved for later versions.

## Build

```bash
go mod tidy
go build -o GoTravel .
```

## Import Gator CSV

```bash
./GoTravel import gator examples/gator/sample.csv
```

Use a custom database:

```bash
./GoTravel import --db gotravel.sqlite gator examples/gator/sample.csv
```

Read from stdin:

```bash
cat examples/gator/sample.csv | ./GoTravel import gator -
```

By default, corrupt input aborts the file import and commits nothing for that source.

Use `--force` to skip corrupt rows, store them in `import_errors`, and commit valid rows:

```bash
./GoTravel import --force gator messy.csv
```

## Export staged CSV

```bash
./GoTravel export output.csv
```

Refuse to overwrite unless `--force` is used:

```bash
./GoTravel export --force output.csv
```

Write to stdout:

```bash
./GoTravel export -
```

Filter by date/time:

```bash
./GoTravel export output.csv --start 2025
./GoTravel export output.csv --start 2025-05
./GoTravel export output.csv --start 2025-05-01 --stop 2025-05-31
./GoTravel export output.csv --start "2025-05-01 10:11:05" --stop "2025-05-01 17:00:00"
```

## Current staged export columns

```csv
dt,lat,lng,altitude,angle,speed,params
```

## Layout

```text
cmd/       CLI commands
examples/  generic examples
export/    exporting code
import/    importing code
profiles/  import/export profile files
routing/   OpenRouteService/OSRM placeholders
storage/   SQLite and file handling
tests/     test framework and fixtures
```
