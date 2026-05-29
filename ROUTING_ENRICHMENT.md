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
- `GoTravel route-match run` wires the runner into the CLI with provider, profile, date filter, and radius options.
- `GoTravel route-match inspect` prints stored route-match run summaries.
- `GoTravel route-match export geojson` exports stored matched geometry when the stored geometry is already GeoJSON.

Not implemented yet:

- No automatic trip segmentation.
- No GPX export of matched geometry.
- No conversion from encoded polyline or provider-specific geometry into GeoJSON.

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

## CLI workflow

```bash
GoTravel route-match run --provider noop --profile driving
GoTravel route-match inspect 1
GoTravel route-match export geojson 1 matched.geojson
```

The run command prints the stored run ID, provider, profile, status, source point count, distance, duration, and geometry format.

## Deferred storage decisions

The current schema intentionally keeps matched geometry and raw response inline with `route_match_runs`.

Revisit this if:

1. `raw_response` becomes large enough to justify a side table.
2. geometry storage needs variants or multiple formats.
3. failed provider calls need to be stored as run records.
4. warnings need queryable rows instead of JSON text.
5. matched runs need explicit immutability/version metadata.

## Non-goals for the current CLI/export work

Do not add:

- automatic trip segmentation
- dwell-time logic
- UI/report generation
- GPX export of matched routes
- provider-specific database tables
- live OSRM integration tests

## Recommended next implementation sequence

1. Manually test route matching with real-ish staged data against a local OSRM instance.
2. Add geometry conversion if a provider stores encoded polyline or another non-GeoJSON geometry format.
3. Add matched GPX export only after stored geometry behaviour is proven useful.
4. Add reporting/trip segmentation later, as separate work.
