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

## Gator Tracker full list

Core Hardware & Signal Elements

Log Param [1, 3, 4, 5]	Teltonika AVL ID	Parameter Description	Expected Values & Units
gsmlev	ID 21	GSM/Cellular network signal strength indicator.	0 (No signal) to 31 (Excellent)
pdop	—	Positional Dilution of Precision (Satellite geometry spread).	Lower is better. < 2.0 is ideal, > 5.0 is poor.
io1	ID 1	Digital Input 1 (Hardwired to vehicle Ignition Status).	0 = Ignition OFF / 1 = Ignition ON
io3	ID 3	Digital Input 3 (Typically hardwired to an optional SOS Button).	0 = Normal / 1 = Triggered Panic
io6	ID 6	Analog Input 1 voltage level.	Millivolts (mV). E.g., 12000 = 12.0V
io11	ID 11	ICCID identification number of your Telstra M2M SIM.	Unique 19 or 20-digit SIM serial number
io14 *	ID 14	Total device Odometer calculated via GPS tracking.	Value in meters (m).


Device States & Diagnostic Data

Log Param [1, 3, 4, 5]	Teltonika AVL ID	Parameter Description	Expected Values & Units
io24 *	ID 24	Device movement state parsed by the internal accelerometer.	0 = Stationary / 1 = In Motion
io66 *	ID 66	External Voltage supplied from the main vehicle battery.	Millivolts (mV). E.g., 13800 = 13.8V
io67	ID 67	Internal Backup Battery Voltage.	Millivolts (mV). Typically 3700 to 4200 (3.7V–4.2V)
io113	ID 113	Internal Backup Battery Level.	Percentage scale: 0 to 100%
io175	ID 175	Auto-Geofence protocol status trigger.	0 = No Event, 1 = Enter, 2 = Exit
io180	ID 180	Bluetooth connection state identifier.	0 = Disconnected / 1 = Connected
io200	ID 200	Internal power-saving Sleep Mode profile status.	0 = Awake, 1 = Deep Sleep, 2 = Ultra Sleep


Security, Crash, & Network Telemetry

Log Param [1, 3, 4]	Teltonika AVL ID	Parameter Description	Expected Values & Units
io236	ID 236	Alarm event flags generated by system monitoring.	Hex codes mapping specific error states
io239 *	ID 239	Ignition On/Off event status changes.	0 = Turned Off / 1 = Turned On
io240 *	ID 240	Movement state change event flag.	0 = Stopped / 1 = Started Moving
io241	ID 241	Active GSM Operator Code identifier (MCC+MNC).	50501 for Telstra
io246 *	ID 246	Total duration of current idling period.	Value captured in seconds (s)
io247	ID 247	Crash Detection Scenario sensor trigger status.	0 = Normal / 1 = Impact event detected
io251 *	ID 251	Idling Status notification flag.	0 = Normal driving/stop / 1 = Engine idling
io252	ID 252	Unplug / Power Cut alarm monitoring state.	1 = External main vehicle power disconnected
io253 *	ID 253	Eco-Driving / Driver Behaviour event code.	1 = Acceleration, 2 = Hard Braking, 3 = Cornering
io254 *	ID 254	Continuous vehicle trip status tracker.	0 = Trip Ended / 1 = Trip Active
io303 *	ID 303	Instant Acceleration forces recorded on the Z-axis.	Measured in milli-G forces (mG)
io310	ID 310	Active ICCID SIM slot indicator selection.	1 = Primary SIM Slot
io380	ID 380	Standard Teltonika firmware system error code flag.	0 = Normal / >0 = Error active
io381	ID 381	Network connection or data socket state identifier.	0 = Data disconnected / 1 = Server online


Accelerometer Vector Axis Raw Data
The final three parameters report the directional gravity forces detected by the tracker's internal 3-axis gyro sensor: [1, 2]
* g0: Raw X-Axis accelerometer force value (Left/Right vector changes).
* g1: Raw Y-Axis accelerometer force value (Forward/Backward vector changes).
* g2: Raw Z-Axis accelerometer force value (Up/Down vector changes). [1, 2]
Would you like help setting up a custom Event Trigger Alert inside your portal for one of these specific parameter IDs (like io247 for impacts or io252 for power cuts)?


