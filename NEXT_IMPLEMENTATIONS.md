# GoTravel Next Implementations

This file tracks near-term implementation direction for GoTravel so future manual work, Codex work, or other AI-agent work does not wander off into the shrubbery and return with a web framework.

## Current Baseline

GoTravel currently has a deliberately simple staged workflow:

1. Initialise and verify a SQLite staging database.
2. Import known CSV formats into SQLite.
3. Preserve source provenance and corrupt-row errors.
4. Skip duplicate points by stable point hash.
5. Export staged data predictably.
6. Add routing and richer reporting only after staging remains reliable.

Current active pieces:

- `GoTravel db init`
- `GoTravel db verify`
- `GoTravel db export <filename>`
- `GoTravel db import <filename>`
- `GoTravel import gator <input.csv> [...]`
- `GoTravel import gator -`
- `GoTravel export gator <output.csv|->`
- `GoTravel export gpx <output.gpx|->`

Current reserved pieces:

- `google` import/export implementation.
- `audit` export.
- KML export.
- Route matching.
- ORS/OSRM integration.
- Trip segmentation.
- Dwell-time calculation.
- HTML maps or reports.
- Web UI.
- Background services.

## Completed Implementation Notes

The following previously planned items have now been implemented and documented:

1. Database command group.
2. Explicit export format command shape.
3. GPX export from staged points.
4. Partial date/time filter precision through `YYYY-MM-DD HH`, `YYYY-MM-DD HH:MM`, and `YYYY-MM-DD HH:MM:SS`.

The old ambiguous export command:

```bash
GoTravel export <output.csv>
```

has been replaced by explicit format commands:

```bash
GoTravel export gator <output.csv|-> [--db gotravel.sqlite] [--force] [--start VALUE] [--stop VALUE]
GoTravel export gpx <output.gpx|-> [--db gotravel.sqlite] [--force] [--start VALUE] [--stop VALUE]
```

`google` remains reserved:

```bash
GoTravel export google <output.csv|->
```

## Next Implementation: Audit Export

### Goal

Add an explicit audit-oriented CSV export that preserves the staged point plus provenance fields and selected parsed tracker parameters.

This is not a replacement for the simple Gator staged CSV export or the GPX export. It is an additional export mode intended for inspection, debugging, duplicate investigation, import/export comparison, and future trip-segmentation work.

### Command

```bash
GoTravel export audit <output.csv|-> [--db gotravel.sqlite] [--force] [--start VALUE] [--stop VALUE]
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
io3
io6
io11
io14
io24
io66
io67
io113
io175
io180
io200
io236
io239
io240
io241
io246
io247
io251
io252
io253
io254
io303
io310
io380
io381
g0
g1
g2
```

Missing params must be exported as empty values, not errors.

### Behaviour Rules

- Reuse existing date filter behaviour.
- Support partial date/time filter precision already supported by `storage.ParsePartialDateTime`.
- Refuse output overwrite unless `--force` is provided.
- Support stdout with `-`.
- Do not alter stored raw `params`.
- Do not interpret movement yet; only expose fields predictably.
- Do not add routing logic.
- Do not call routing providers.

### Package Boundaries

- CLI wiring belongs in `cmd/`.
- CSV generation belongs in `export/`.
- Database reads and query helpers belong in `storage/`.
- Tracker param parsing helpers may live in `export/`, `import/`, or `storage/` only if they remain simple and deterministic.

### Tests

Add tests for:

- `GoTravel export audit <file>`.
- `GoTravel export audit -`.
- Date filtering.
- Partial time filtering.
- Overwrite refusal.
- Required provenance columns.
- Tracker param expansion.
- Empty params.
- Unknown/malformed params remaining harmless.

### Documentation

When implemented, update:

- `COMMANDS.md`
- `README.md`
- `CHANGES.md`

## Later Candidate: Google Import/Export

### Goal

Add the Google CSV import/export shape only after Gator import/export, GPX export, and audit export are stable.

### Reserved Commands

```bash
GoTravel import google <input.csv> [...]
GoTravel export google <output.csv|-> [--db gotravel.sqlite] [--force] [--start VALUE] [--stop VALUE]
```

### Behaviour Rules

- Do not auto-detect Google CSVs.
- Keep format explicit.
- Preserve source provenance.
- Preserve duplicate detection rules.
- Add fixtures before implementation is considered complete.

## Explicitly Not In The Next Implementation

These are out of scope for the next implementation step unless explicitly requested:

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
- Existing export behaviour is preserved under explicit commands.
- No protected files have been modified.
