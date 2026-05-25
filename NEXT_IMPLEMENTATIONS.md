# GoTravel Next Implementations

This file locks down the next implementation steps for GoTravel so future manual work, Codex work, or other AI-agent work does not wander off into the shrubbery and return with a web framework.

## Current Baseline

GoTravel currently prioritises a deliberately simple staged workflow:

1. Import known CSV formats into SQLite.
2. Preserve source provenance and corrupt-row errors.
3. Export staged data predictably.
4. Add routing and richer reporting only after staging is reliable.

The current active importer is `gator`. The `google` importer, GPX/KML export, route analysis, reports, and maps remain reserved until explicitly implemented.

## Implementation 1: Database Command Group

### Goal

Add an explicit `db` command group for database lifecycle and backup/restore style operations.

This should be easy, quick, and boring. Boring is good. Boring tools are the ones that do not wake you at 2:00am wearing a hat made of stack traces.

### Commands

```bash
GoTravel db init [--db gotravel.sqlite] [--force]
GoTravel db export [--db gotravel.sqlite] [--force] <filename>
GoTravel db import [--db gotravel.sqlite] [--force] <filename>
```

### Behaviour Requirements

#### `GoTravel db init`

- Create the SQLite database if it does not exist.
- Create required schema tables if missing.
- Be safe to run repeatedly when the schema is already present.
- Refuse destructive reinitialisation unless `--force` is explicitly supplied.
- Do not import data.
- Do not export data.

#### `GoTravel db export <filename>`

- Export the whole SQLite database to a file.
- Intended use is backup, transfer, and inspection.
- Refuse to overwrite `<filename>` unless `--force` is supplied.
- Do not apply GPS date filters.
- Do not transform rows.
- Do not emit staged CSV; this is database export, not point export.

#### `GoTravel db import <filename>`

- Import/restore a database file into the configured database path.
- Refuse to overwrite an existing database unless `--force` is supplied.
- Validate that `<filename>` is a usable SQLite database before replacing the target where practical.
- Do not merge rows.
- Do not transform rows.
- Keep this as a whole-database restore/import operation.

### Package Boundaries

- CLI wiring belongs in `cmd/`.
- Database creation, schema checks, backup, restore, and validation belong in `storage/`.
- File overwrite safety helpers should remain shared and explicit.

### Tests

Add tests for:

- Init creates required schema.
- Init is safe when run repeatedly.
- Init refuses destructive work without `--force`.
- DB export refuses overwrite without `--force`.
- DB export creates a usable SQLite database copy.
- DB import refuses to overwrite an existing target without `--force`.
- DB import rejects clearly invalid files.
- DB import restores a usable database.

### Documentation

When implemented, update:

- `COMMANDS.md`
- `README.md`
- `CHANGES.md`

## Implementation 2: Export Command Namespace Restructure

### Goal

Restructure export commands so export format is explicit.

The current staged CSV export command:

```bash
GoTravel export <output.csv>
```

must become a format-specific command:

```bash
GoTravel export gator [--db gotravel.sqlite] [--force] <output.csv|-> [--start VALUE] [--stop VALUE]
GoTravel export google [--db gotravel.sqlite] [--force] <output.csv|-> [--start VALUE] [--stop VALUE]
```

`gator` is active first. `google` is reserved until the Google importer/export shape is explicitly implemented.

### Behaviour Requirements

- `GoTravel export gator <output.csv|->` replaces the original `GoTravel export <output.csv|->` behaviour.
- Preserve the current Gator staged CSV columns unless explicitly changed:

```csv
dt,lat,lng,altitude,angle,speed,params
```

- Preserve existing date filter support:

```text
YYYY
YYYY-MM
YYYY-MM-DD
YYYY-MM-DD HH:MM:SS
```

- Preserve overwrite safety: refuse existing output files unless `--force` is supplied.
- Preserve stdout behaviour with `-`.
- Do not silently keep the old ambiguous command as the primary documented interface.
- If backward compatibility is kept temporarily, it must warn clearly that `GoTravel export <output.csv>` is deprecated.

### Package Boundaries

- CLI routing belongs in `cmd/`.
- Format-specific CSV export logic belongs in `export/`.
- Database reads and date filtering belong in `storage/`.

### Tests

Add or update tests for:

- `GoTravel export gator <file>`.
- `GoTravel export gator -`.
- Date filtering under the new command shape.
- Overwrite refusal under the new command shape.
- Unknown export format errors clearly.
- Deprecated old export syntax, if temporarily retained.

### Documentation

When implemented, update:

- `COMMANDS.md`
- `README.md`
- `CHANGES.md`

## Implementation 3: Audit Export

### Goal

Add an explicit audit-oriented CSV export that preserves the staged point plus provenance fields and selected parsed tracker parameters.

This is not a replacement for the simple Gator/Google staged CSV export. It is an additional export mode intended for inspection, debugging, and future trip-segmentation work.

### Command

```bash
GoTravel export audit [--db gotravel.sqlite] [--force] <output.csv|-> [--start VALUE] [--stop VALUE]
```

### Output Requirements

The audit export should include, at minimum:

```text
dt
lat
lng
altitude
angle
speed
params
format
source_file
source_line
imported_at
point_hash
```

It should also expand known tracker params into stable columns where available:

```text
gpslev
gsmlev
pdop
io1
io14
io24
io66
io67
io113
io175
io200
io239
io240
io246
io247
io251
io252
io253
io254
io303
io380
io381
g0
g1
g2
```

Missing params must be exported as empty values, not errors.

### Behaviour Rules

- Reuse existing date filter behaviour.
- Refuse output overwrite unless `--force` is provided.
- Support stdout with `-`.
- Do not alter stored raw `params`.
- Do not interpret movement yet; only expose fields predictably.
- Add tests for stdout export, file export, date filtering, overwrite refusal, and param expansion.
- Update `COMMANDS.md`, `README.md`, and `CHANGES.md` when implemented.

### Package Boundaries

- CLI wiring belongs in `cmd/`.
- CSV generation belongs in `export/`.
- Database reads and query helpers belong in `storage/`.
- Tracker param parsing helpers may live in `import/` or `storage/` only if they remain simple and deterministic.

## Implementation 4: GPX Export From Staged Points

### Goal

Add GPX export from already-staged points.

This should convert staged point data into a predictable GPX track without introducing routing, map matching, trip segmentation, ORS, OSRM, or other machinery. It is a file format export, not a journey oracle wearing a false moustache.

### Command

```bash
GoTravel export gpx [--db gotravel.sqlite] [--force] <output.gpx|-> [--start VALUE] [--stop VALUE]
```

### Output Requirements

- Generate GPX 1.1.
- Emit one track containing one segment ordered by GPS timestamp.
- Preserve timestamp as GPX point time.
- Use latitude and longitude as GPX point attributes.
- Include elevation only when altitude is present and valid.
- Do not invent routes, stops, names, or dwell times.
- Do not call routing providers.

### Behaviour Rules

- Reuse existing date filter behaviour.
- Refuse output overwrite unless `--force` is provided.
- Support stdout with `-`.
- Return a clear error when there are no matching points.
- Add tests for basic GPX structure, ordering, stdout export, date filtering, overwrite refusal, and empty result handling.
- Update `COMMANDS.md`, `README.md`, and `CHANGES.md` when implemented.

### Package Boundaries

- CLI wiring belongs in `cmd/`.
- GPX formatting belongs in `export/`.
- Database reads and filtering belong in `storage/`.
- Do not add routing logic.
- Do not add a GPX dependency unless the standard library approach becomes genuinely painful.

## Explicitly Not In These Implementations

These are out of scope for the next implementation steps:

- Google CSV import, except reserving/documenting the `export google` command shape.
- KML export.
- Route matching.
- ORS integration.
- OSRM integration.
- Trip segmentation.
- Dwell-time calculation.
- HTML maps or reports.
- Web UI.
- Background services.
- Database replacement.

## Suggested Order

1. Implement `GoTravel db init`, `GoTravel db export <filename>`, and `GoTravel db import <filename>` first.
2. Restructure staged CSV export from `GoTravel export <output.csv>` to `GoTravel export gator <output.csv>` and reserve `GoTravel export google <output.csv>`.
3. Implement `GoTravel export audit <output.csv|->`.
4. Implement `GoTravel export gpx <output.gpx|->`.
5. Revisit Google import, route analysis, KML, maps, and reports only after these steps are tested and documented.

## Acceptance Checklist

Before any implementation is considered complete:

```bash
gofmt -w .
go test ./...
```

Also check:

- Behaviour is covered by tests.
- Documentation matches command behaviour.
- `CHANGES.md` records the change.
- Existing import behaviour has not changed unintentionally.
- Existing export behaviour is either preserved under the new explicit command or clearly deprecated.
- No protected files have been modified.
