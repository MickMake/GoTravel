#!/bin/bash

GoTravel route-match run \
	--provider osrm \
	--profile driving \
	--osrm-base-url http://127.0.0.1:5000 \
	--from "2025-05-02 00:00" \
	--to "2025-05-02 21:30" \
	| tee matched-osrm.txt
ID="$(awk -F= '/route_match_run_id/{print$2}' matched-osrm.txt)"
rm matched-osrm.gpx
GoTravel route-match export gpx ${ID} matched-osrm.gpx

