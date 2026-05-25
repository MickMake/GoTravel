# GoTravel Next Implementations

This file locks down the next implementation steps for GoTravel so future manual work, Codex work, or other AI-agent work does not wander off into the shrubbery and return with a web framework.

## Current Baseline

GoTravel currently prioritises a deliberately simple staged workflow:

1. Import known CSV formats into SQLite.
2. Preserve source provenance and corrupt-row errors.
3. Export staged data predictably.
4. Add routing and richer reporting only after staging is reliable.

The current active importer is `gator`. The `google` importer, GPX/KML export, route analysis, reports, and maps remain reserved until explicitly implemented.

## Implementation 1: Expand Staged Export / Audit CSV

### Goal

Add an explicit audit-oriented CSV export that preserves the staged point plus provenance fields and selected parsed tracker parameters.

This is not a replacement for the current simple export. It is an additional export mode intended for inspection, debugging, and future trip-segmentation work.

### Proposed Command

```bash
GoTravel export-audit [--db gotravel.sqlite] [--force] <output.csv|-> [--start VALUE] [--stop VALUE]
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

- Keep the current `GoTravel export` command unchanged.
- Reuse existing date filter behaviour.
- Refuse output overwrite unless `--force` is provided.
- Do not alter stored raw `params`.
- Do not interpret movement yet; only expose fields predictably.
- Add tests for stdout export, file export, date filtering, overwrite refusal, and param expansion.
- Update `COMMANDS.md`, `README.md`, and `CHANGES.md` when implemented.

### Package Boundaries

- CLI wiring belongs in `cmd/`.
- CSV generation belongs in `export/`.
- Database reads and query helpers belong in `storage/`.
- Tracker param parsing helpers may live in `import/` or `storage/` only if they remain simple and deterministic.

## Implementation 2: GPX Export From Staged Points

### Goal

Add GPX export from already-staged points.

This should convert staged point data into a predictable GPX track without introducing routing, map matching, trip segmentation, ORS, OSRM, or other machinery. It is a file format export, not a journey oracle wearing a false moustache.

### Proposed Command

```bash
GoTravel export-gpx [--db gotravel.sqlite] [--force] <output.gpx|-> [--start VALUE] [--stop VALUE]
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

These are out of scope for the next two implementation steps:

- Google CSV import.
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

1. Implement `export-audit` first because it exposes the raw evidence needed to validate movement and tracker signals.
2. Implement `export-gpx` second because it provides a useful external format while still relying only on staged points.
3. Revisit Google import, route analysis, and reports only after these exports are tested and documented.

## Acceptance Checklist

Before either implementation is considered complete:

```bash
gofmt -w .
go test ./...
```

Also check:

- Behaviour is covered by tests.
- Documentation matches command behaviour.
- `CHANGES.md` records the change.
- Existing `import` and `export` behaviour has not changed unintentionally.
- No protected files have been modified.
