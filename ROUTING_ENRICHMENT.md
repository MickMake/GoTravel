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
- `route_match_runs` stores provider-neutral route match run results.
- `route_match_points` links persisted route match runs back to source point IDs in sequence.
- `storage.RouteMatchRunner` loads staged points, builds a trace request, calls the enricher, converts the result, and persists the route match run.

Not implemented yet:

- No CLI route-matching command.
- No export/report of matched route data.
- No automatic trip segmentation.
- No GPX export of matched geometry.

## Data flow

```text
storage.Point rows
  -> storage.MatchTraceRequestFromPoints
  -> routing.MatchTraceRequest
  -> routing.Enricher.MatchTrace
  -> routing.MatchTraceResult
  -> routing.EnrichedTrace
  -> storage.SaveRouteMatchRun
  -> route_match_runs + route_match_points
```

Provider-specific code stays behind `routing.Provider` implementations.

## Provider-neutral enriched trace fields

The in-memory model captures:

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

## Persisted route match data

Persistence uses explicit route-match run records instead of silently mutating imported GPS points.

### `route_match_runs`

One row per provider call/batch.

Fields:

- `id`
- `provider`
- `profile`
- `status`
- `source_point_count`
- `geometry`
- `geometry_format`
- `distance_meters`
- `duration_seconds`
- `confidence`
- `warnings_json`
- `raw_response`
- `matched_at`
- `created_at`
- `source_filter_start`
- `source_filter_end`

### `route_match_points`

Join table linking matched runs back to source points.

Fields:

- `id`
- `route_match_run_id`
- `point_id`
- `sequence`

## Deferred storage decisions

The current schema intentionally keeps matched geometry and raw response inline with `route_match_runs`.

Revisit this if:

1. `raw_response` becomes large enough to justify a side table.
2. geometry storage needs variants or multiple formats.
3. failed provider calls need to be stored as run records.
4. warnings need queryable rows instead of JSON text.
5. matched runs need explicit immutability/version metadata.

## Non-goals for the next CLI PR

Do not add:

- automatic trip segmentation
- dwell-time logic
- UI/report generation
- GPX export of matched routes
- provider-specific database tables
- live OSRM integration tests

## Recommended next implementation sequence

1. Add a CLI route-match command that uses the existing internal runner.
2. Add clear command documentation and safety notes.
3. Test with real-ish staged data against a local OSRM instance manually, outside unit tests.
4. Add export/report features only after persisted route match runs are proven useful.
