#!/bin/bash

go run . route-match run   --provider osrm   --profile driving   --osrm-base-url http://127.0.0.1:5000   --from "2025-05-02 00:00"   --to "2025-05-02 21:30"
rm matched.gpx
go run . route-match export gpx 1 matched.gpx
