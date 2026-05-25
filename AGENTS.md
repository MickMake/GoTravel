# Agent Instructions for GoTravel

This file applies to any AI/code agent working on GoTravel.

## Project Identity

GoTravel is a simple CLI tool for tracker/GPS data processing.

Current priority:

1. Import known CSV formats into SQLite.
2. Preserve source provenance and errors.
3. Export staged data predictably.
4. Add routing/reporting later only after staging is reliable.

## Non-Negotiables

- Keep the command-line workflow simple.
- Keep SQLite as the staging database.
- Keep importers/exporters explicit.
- Preserve source file and line metadata.
- Abort on corrupt input unless `--force` is used.
- Refuse output overwrite unless `--force` is used.
- Update documentation when behaviour changes.

## Package Boundaries

```text
cmd/       CLI wiring only
import/    input format parsing only
export/    output generation only
storage/   database, schema, file safety, models
routing/   future route provider abstraction
profiles/  import/export mappings and documentation
examples/  small samples
tests/     fixtures and tests
```

Do not move logic into `cmd/` if it belongs in `import/`, `export/`, or `storage/`.

## Design Preference

Prefer:

- Short functions.
- Explicit structs.
- Standard library first.
- Small interfaces.
- Clear errors.
- Table-driven tests.

Avoid:

- Global mutable state.
- Reflection-heavy parsing.
- Magic auto-detection.
- Broad rewrites.
- Hidden network calls.
- Unnecessary dependencies.

## Data Handling Rules

Every imported point must preserve:

```text
GPS timestamp
latitude
longitude
optional telemetry fields
source format
source file
source line
import timestamp
stable point hash
```

Corrupt rows must be traceable to:

```text
import run
source file
source line
raw row
error message
```

## Routing Rules

The folder is named `routing/`, not `ors/`.

Routing providers must be pluggable:

- OpenRouteService
- OSRM

Do not hard-code routing assumptions into import/export/storage.

## Before Submitting Changes

Check:

```bash
gofmt -w .
go test ./...
```

Then update documentation if needed.

If either command cannot be run, report that honestly.

## Behaviour Preservation

Refactors must not change behaviour unless the task explicitly asks for behaviour change.

If behaviour changes accidentally, either revert it or document it and confirm it is desired.
