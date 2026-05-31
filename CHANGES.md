# Changelog

## Unreleased

- Added provider-neutral route-match trace hygiene to remove exact consecutive duplicate coordinates before provider calls while preserving staged point order and source point links.
- Added conservative route-match trace chunking for long traces using 100-point chunks with one-point overlap.
- Combined successful route-match chunks into one stored run with summed distance/duration, combined GeoJSON LineString geometry, and per-chunk raw provider responses stored as a JSON array.
- Added route-match trace hygiene/chunking tests covering duplicate removal, chunk limits, overlap behaviour, combined geometry/metrics, and chunk failure storage safety.
- Added OpenRouteService-local routing profile aliases for common neutral profile names while preserving ORS-native profile values unchanged.
- Implemented the OpenRouteService HTTP provider client for route and matrix operations using provider-neutral result types and preserved raw responses.
- Kept OpenRouteService health, snapping, and trace matching conservative/unimplemented until verified endpoint workflows are available; ORS directions remains route-through-waypoints, not trace/map matching.
- Added minimal OpenRouteService provider factory config for base URL and profile selection.
- Added `httptest` coverage for OpenRouteService defaults, capabilities, provider factory wiring, route request construction, GeoJSON response mapping, matrix request construction, matrix dimensions, raw response preservation, provider errors, HTTP errors, and explicit unimplemented snap/trace matching.
- Implemented the Valhalla HTTP provider client for health, route, trace matching, snapping, and matrix operations using provider-neutral result types and preserved raw responses.
- Added minimal Valhalla provider factory config for base URL and profile selection.
- Added `httptest` coverage for Valhalla request construction, response parsing, raw response preservation, validation, matrix dimensions, provider status errors, and HTTP errors without requiring a live Valhalla server.
- Addressed PR review follow-ups for routing registry initialisation, route-match coordinate validation, SQLite foreign-key enforcement, noop route matching, and route-match export preflight validation.
- Fixed route geometry conversion to reuse the shared routing coordinate type.
- Added provider-neutral route geometry conversion for stored GeoJSON, encoded polyline precision 5, and encoded polyline precision 6.
- Updated `GoTravel route-match export geojson` to convert supported encoded polyline geometry into GeoJSON LineString output.
- Added `GoTravel route-match export gpx` to export stored matched route geometry as a GPX 1.1 track.
- Added route geometry conversion and matched route export tests without requiring a live OSRM server.
- Added `GoTravel route-match run` to route-match staged points through the existing provider-neutral runner.
- Added route-match provider/profile/date-filter/radius CLI options, including OSRM base URL wiring through the provider factory.
- Added `GoTravel route-match inspect` for stored route-match run summaries.
- Added `GoTravel route-match export geojson` for stored matched geometry that is already GeoJSON.
- Added route-match CLI helper tests without requiring a live OSRM server.
- Documented route-match commands and updated routing enrichment status.
- Added OSRM provider usage examples and an interface conformance check.
- Added a provider-neutral routing framework with providers (`noop`, `ors`, `osrm`, `valhalla`), shared routing contracts/types, and registry/tests.
- Implemented the OSRM HTTP provider client for health, route, trace matching, snapping, and matrix operations using provider-neutral result types and preserved raw responses.
- Hardened OSRM provider validation for route, trace-matching, and matrix requests, including matrix response dimension checks.
- Added `httptest` coverage for OSRM URL construction, response parsing, raw response preservation, request validation, matrix dimensions, and error handling without requiring a live OSRM server.
- Documented Valhalla as a planned routing provider alongside OpenRouteService and OSRM.
- Clarified that the core routing interface should expose only operations shared by all supported routing providers.

## 0.5

- Added `GoTravel export gpx <output.gpx|->` to export staged points as GPX 1.1.
- GPX export writes one track containing one segment ordered by staged point timestamp.
- GPX export does not perform route matching, trip segmentation, dwell-time calculation, or provider calls.
- Added partial time precision for export date filters: `YYYY-MM-DD HH`, `YYYY-MM-DD HH:MM`, and `YYYY-MM-DD HH:MM:SS`.
- Documented GPX export and partial time filters in `COMMANDS.md` and `README.md`.

## 0.4

- Added `GoTravel db init` to initialise the SQLite database and required schema.
- Added `GoTravel db verify` to validate that a configured database is usable and has required GoTravel tables.
- Added `GoTravel db export <filename>` to copy the whole SQLite database to a backup/transfer file.
- Added `GoTravel db import <filename>` to restore a whole GoTravel SQLite database file.
- Added database validation for import/export operations.
- Documented database commands in `COMMANDS.md` and `README.md`.

## 0.3

- Added `AUTHORITATIVE_SPECIFICATION.md` as the top-level behavioural specification.
- Added `COMMANDS.md` to lock down current CLI syntax and safety rules.
- Added `CODEX.md` to constrain future Codex work.
- Added `AGENTS.md` for general AI/code-agent workflow rules.
- Added updated `README.md` aligned with the current staged import/export scope.
- Added `TRACKER_SIGNALS.md.template` to avoid overwriting an existing tracker-signal specification.

## 0.2

- Split GoTravel into `cmd`, `import`, `export`, `routing`, `profiles`, and `storage` packages.
- Renamed the routing concept to `routing` instead of `ors`.
- Added canonical internal `Point` model.
- Added source metadata: `source_file`, `source_line`, and `imported_at`.
- Added `import_runs` table.
- Added `import_errors` table.
- Added `--force` support for imports: skip corrupt rows, store errors, and commit valid rows.
- Added default import safety: corrupt rows abort the file import unless `--force` is used.
- Added default export safety: existing output files are not overwritten unless `--force` is used.
- Preserved staged CSV export behaviour.
- Added basic tests and fixtures.

## 0.1

- Initial simple staged import/export implementation.
- Added Gator CSV import into SQLite.
- Added staged CSV export from SQLite.
- Added date/time filtering for export.
