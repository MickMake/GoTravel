<!--
PROTECTED FILE - DO NOT EDIT

This file is intentionally protected.
Do not modify, rewrite, reformat, rename, move, or delete this file.

Codex/AI agents:
- You must not change this file.
- If a task appears to require changing this file, stop and report that it is protected.
- Do not "clean up", "modernise", "simplify", or "deduplicate" this file.
-->

# Tracker Signals and Params Reference

This file captures raw tracker signal context that should not be lost during future Codex or manual maintenance work.

## Why raw GPS is canonical

GoGator exists because Gator's processed "drives and stops" exports have shown suspicious timestamp and trip-detection behaviour. Those processed exports are useful as a comparison point, but they are not the truth source.

The raw GPS CSV is treated as canonical because it preserves the original tracker observations: timestamps, noisy coordinates, speeds, odometer-like values, movement states, idling hints, GPS quality, and accelerometer/crash/driving-style fields. GoGator should enrich those observations deterministically using local reference files, not replace them with opaque vendor processing.

## Raw CSV columns

Raw GPS rows are expected as either headed or headerless CSV:

```text
dt,lat,lng,altitude,angle,speed,params
```

If the file is headerless, assume that exact order. Raw row numbers must match source file line numbers.

## Params format

The `params` field is unordered key/value data. Parse it into stable columns regardless of the order provided by the tracker.

Known params currently include:

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

## Key movement and quality signals

- `io24`: movement state. `0` is strong stationary evidence. `1` is strong movement evidence.
- `io251`: idling status. `1` is idling/stationary evidence. `0` is not proof of movement.
- `pdop`: GPS geometry/quality hint. Useful, but not enough by itself; a bad coordinate can still have good PDOP.
- `io14`: odometer in metres where available. Useful for detecting false stationary teleports.
- `gpslev`: GPS satellite/level signal where available.
- `gsmlev`: GSM signal level where available.

## Accelerometer fields

Preserve these fields in expanded/audit/signal summary output where applicable:

- `g0`: raw X-axis acceleration, left/right vector
- `g1`: raw Y-axis acceleration, forward/back vector
- `g2`: raw Z-axis acceleration, up/down vector

These are useful for diagnosing harsh movement, vibration, crash-like behaviour, false clusters, and general tracker goblin activity.

## Crash and driving-style fields

Preserve these fields where applicable:

- `io247`
- `io253`
- `io303`
- `g0`
- `g1`
- `g2`

Do not discard these just because they are not always needed for trip segmentation. They are part of the raw evidence trail.

## Interpretation rules

- Movement detection must not rely on speed alone.
- Combine speed, `io24`, idling, ignition-like signals, odometer, accelerometer values, and GPS quality.
- Treat `io24=0` as strong stationary evidence.
- Treat `io24=1` as strong movement evidence.
- Treat `io251=1` as idling/stationary evidence.
- Do not treat `io251=0` as proof that the vehicle moved.
- Treat PDOP as a quality hint, not a veto or guarantee.
- Keep the raw observed/noisy GPS coordinates in processed output, even when a known site matches.

## Debugging value

The observed GPS coordinate that matched a site is valuable because it shows exactly what the tracker reported and what GoGator accepted. Replacing it with the canonical site coordinate would hide useful debugging evidence and duplicate information already present in `sites.csv`.
