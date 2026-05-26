# GoTravel Routing Provider Contract

This file defines the planned routing provider contract for GoTravel.

Routing support is future work. This document exists to prevent provider-specific assumptions from leaking into import, export, storage, or reporting code before routing is implemented.

## Scope

Supported planned providers:

- OpenRouteService
- OSRM
- Valhalla

The core GoTravel routing interface must expose only operations shared by all supported providers.

Provider-specific features must stay behind optional extensions, provider-specific packages, or raw provider responses until explicitly approved.

## Non-goals for the core interface

Do not add these to the core provider interface yet:

- Isochrones
- Elevation sampling
- Route optimisation / travelling-salesman behaviour
- Detailed turn-by-turn instruction models
- Provider-specific costing knobs
- Traffic/live data
- Provider-specific metadata fields except raw provider response storage

These may be considered later as optional extensions after the staged import/export path and basic route matching remain stable and tested.

## Core interface

The planned Go interface is:

```go
type Provider interface {
    Name() string
    Health(ctx context.Context) error
    Capabilities(ctx context.Context) Capabilities

    Route(ctx context.Context, req RouteRequest) (RouteResult, error)
    MatchTrace(ctx context.Context, req MatchTraceRequest) (MatchTraceResult, error)
    Snap(ctx context.Context, req SnapRequest) (SnapResult, error)
    Matrix(ctx context.Context, req MatrixRequest) (MatrixResult, error)
}
```

## Core capabilities

```go
type Capabilities struct {
    Route      bool
    MatchTrace bool
    Snap       bool
    Matrix     bool
}
```

## Core provider calls

| Provider | API call | Expected input | Expected output |
|---|---|---|---|
| OpenRouteService | `Health` | Base URL / provider config | Provider reachable/readiness status, error if unavailable |
| OSRM | `Health` | Base URL / provider config | Provider reachable/readiness status, error if unavailable |
| Valhalla | `Health` | Base URL / provider config | Provider reachable/readiness status, error if unavailable |
| OpenRouteService | `Capabilities` | Provider config/version info if available | Booleans for common operations: `Route`, `MatchTrace`, `Snap`, `Matrix` |
| OSRM | `Capabilities` | Provider config/version info if available | Booleans for common operations: `Route`, `MatchTrace`, `Snap`, `Matrix` |
| Valhalla | `Capabilities` | Provider config/version info if available | Booleans for common operations: `Route`, `MatchTrace`, `Snap`, `Matrix` |
| OpenRouteService | `Route` | Profile, start coordinate, end coordinate | Status, geometry, geometry format, distance metres, duration seconds, warnings, raw response |
| OSRM | `Route` | Profile, start coordinate, end coordinate | Status, geometry, geometry format, distance metres, duration seconds, warnings, raw response |
| Valhalla | `Route` | Profile, start coordinate, end coordinate | Status, geometry, geometry format, distance metres, duration seconds, warnings, raw response |
| OpenRouteService | `MatchTrace` | Profile, timestamped GPS points, optional accuracy/radius hints | Status, matched geometry, geometry format, distance metres, duration seconds, confidence if available, warnings, raw response |
| OSRM | `MatchTrace` | Profile, timestamped GPS points, optional accuracy/radius hints | Status, matched geometry, geometry format, distance metres, duration seconds, confidence if available, warnings, raw response |
| Valhalla | `MatchTrace` | Profile, timestamped GPS points, optional accuracy/radius hints | Status, matched geometry, geometry format, distance metres, duration seconds, confidence if available, warnings, raw response |
| OpenRouteService | `Snap` | Profile, one or more coordinates | Status, snapped coordinates, warnings, raw response |
| OSRM | `Snap` | Profile, one or more coordinates | Status, snapped coordinates, warnings, raw response |
| Valhalla | `Snap` | Profile, one or more coordinates | Status, snapped coordinates, warnings, raw response |
| OpenRouteService | `Matrix` | Profile, source coordinates, destination coordinates | Status, duration matrix, distance matrix if available, warnings, raw response |
| OSRM | `Matrix` | Profile, source coordinates, destination coordinates | Status, duration matrix, distance matrix if available, warnings, raw response |
| Valhalla | `Matrix` | Profile, source coordinates, destination coordinates | Status, duration matrix, distance matrix if available, warnings, raw response |

## Normalisation rules

GoTravel owns:

- raw imported points
- database storage
- provenance
- segmentation decisions
- reporting decisions

Routing providers own:

- route calculation
- road snapping
- trace matching
- distance/duration enrichment

Provider adapters must translate provider-specific request/response formats into GoTravel's neutral request/result types.

Raw provider responses should be preserved in result structs or storage rows where practical, so future versions can inspect provider-specific details without changing the core contract.

## Stub rules

Initial provider implementations may be stubs.

Stubs must:

- compile
- expose stable provider names
- return conservative capability flags
- return an explicit not-implemented error for unimplemented operations
- make no hidden network calls
- avoid adding new runtime dependencies unless explicitly approved

Expected provider names:

```text
ors
osrm
valhalla
```
