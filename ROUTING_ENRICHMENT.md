# Routing Enrichment Design

## Purpose

Routing enrichment turns imported/staged GPS points into provider-neutral route intelligence.

The first enrichment operation is trace matching: take timestamped GPS points, ask a routing provider to match them to a road network, and keep enough result data for later reporting/export without binding GoTravel to one provider.

## Current scope

Implemented building blocks:

- `routing.Provider` defines provider-neutral routing operations.
- `routing.Service` wraps a provider.
- `routing.Enricher` is the internal enrichment entry point.
- `storage.MatchTraceRequestFromPoints` converts staged `storage.Point` values into `routing.MatchTraceRequest`.
- `routing.EnrichedTrace` captures provider-neutral matched trace output in memory.

Not implemented yet:

- No CLI route-matching command.
- No SQLite persistence for matched routes.
- No route-match run history table.
- No export/report of matched route data.
- No automatic trip segmentation.

## Data flow

```text
storage.Point rows
  -> storage.MatchTraceRequestFromPoints
  -> routing.MatchTraceRequest
  -> routing.Enricher.MatchTrace
  -> routing.MatchTraceResult
  -> routing.EnrichedTrace
```

Provider-specific code stays behind `routing.Provider` implementations.

## Provider-neutral enriched trace fields

The in-memory model should capture:

- provider name
- profile
- status
- source point count
- geometry
- geometry format
- distance in metres
- duration in seconds
- optional confidence
- warnings
- raw provider response
- matched timestamp

These fields are intentionally close to `routing.MatchTraceResult`, but include GoTravel metadata such as source point count and matched time.

## Future persistence goals

When persistence is added, prefer explicit route-match run records instead of silently mutating imported GPS points.

Likely tables:

### `route_match_runs`

One row per provider call/batch.

Candidate fields:

- `id`
- `provider`
- `profile`
- `status`
- `source_point_count`
- `geometry_format`
- `distance_meters`
- `duration_seconds`
- `confidence`
- `matched_at`
- `created_at`
- `source_filter_start`
- `source_filter_end`
- `warnings_json`
- `raw_response`

### `route_match_points`

Optional join table linking matched runs back to source points.

Candidate fields:

- `route_match_run_id`
- `point_id`
- `sequence`

### `route_match_geometry`

Optional split table if geometry becomes large or needs variants.

Candidate fields:

- `route_match_run_id`
- `geometry`
- `geometry_format`

## Open decisions before schema work

Before creating tables, decide:

1. Whether `raw_response` belongs in the main run table or a side table.
2. Whether geometry should be stored inline or separately.
3. Whether source points are linked by explicit point IDs, import-run filters, or both.
4. Whether failed provider calls should be stored as run records.
5. Whether warnings should be JSON text or a separate table.
6. Whether matched runs should be immutable once created.

## Non-goals for the next schema PR

Do not add:

- automatic trip segmentation
- dwell-time logic
- UI/report generation
- GPX export of matched routes
- provider-specific database tables
- live OSRM integration tests

## Recommended next implementation sequence

1. Add schema documentation updates based on this design.
2. Add SQLite tables and migration/initialisation support.
3. Add storage functions for inserting and reading route match runs.
4. Add internal command/service code to run match enrichment over selected staged points.
5. Add CLI only after the storage path is tested.
6. Add export/report features after persisted results exist.
