# GoTravel 0.1

Small staged GPS import/export tool.

## Commands

```bash
GoTravel import [--db gotravel.sqlite] <gator|google> <input.csv> [...]
GoTravel import [--db gotravel.sqlite] <gator|google> -
GoTravel export [--db gotravel.sqlite] <output.csv|-> [--start value] [--stop value]
```

## Current scope

- `gator` import implemented.
- `google` import is reserved but not implemented.
- Import writes normalised GPS rows into SQLite.
- Export writes staged SQLite rows as CSV.
- Date filters accept:
  - `YYYY`
  - `YYYY-MM`
  - `YYYY-MM-DD`
  - `YYYY-MM-DD HH:MM:SS`

## Gator columns expected

```csv
dt,lat,lng,altitude,angle,speed,params
```

## Build

```bash
go mod tidy
go build -o GoTravel .
```
