#!/bin/bash

GoTravel route-match run \
	--provider valhalla \
	--profile auto \
	--from "2025-05-02 09:00" \
	--to "2025-05-03 21:30" \
	| tee matched-valhalla.txt
ID="$(awk -F= '/route_match_run_id/{print$2}' matched-valhalla.txt)"
rm matched-valhalla.gpx
GoTravel route-match export gpx ${ID} matched-valhalla.gpx

