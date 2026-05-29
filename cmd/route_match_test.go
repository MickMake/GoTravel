package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/MickMake/GoTravel/routing"
	"github.com/MickMake/GoTravel/storage"
)

func TestParseOptionalFloat(t *testing.T) {
	value, err := parseOptionalFloat("12.5")
	if err != nil {
		t.Fatalf("parseOptionalFloat returned error: %v", err)
	}
	if value == nil || *value != 12.5 {
		t.Fatalf("parseOptionalFloat value = %v, want 12.5", value)
	}
	if value, err := parseOptionalFloat(""); err != nil || value != nil {
		t.Fatalf("empty radius = %v, %v; want nil, nil", value, err)
	}
	if _, err := parseOptionalFloat("-1"); err == nil {
		t.Fatal("negative radius returned nil error")
	}
}

func TestParseRunID(t *testing.T) {
	id, err := parseRunID("42")
	if err != nil {
		t.Fatalf("parseRunID returned error: %v", err)
	}
	if id != 42 {
		t.Fatalf("parseRunID = %d, want 42", id)
	}
	if _, err := parseRunID("0"); err == nil {
		t.Fatal("zero run ID returned nil error")
	}
}

func TestPrintRouteMatchSummary(t *testing.T) {
	run := storage.RouteMatchRun{
		ID: 7,
		Trace: routing.EnrichedTrace{
			Provider:        "noop",
			Profile:         "driving",
			Status:          "ok",
			SourcePointCount: 3,
			DistanceMeters:  12.3456,
			DurationSeconds: 78.9,
			GeometryFormat:  "geojson",
		},
	}
	var buf bytes.Buffer
	printRouteMatchSummary(&buf, run)
	out := buf.String()
	for _, want := range []string{
		"route_match_run_id=7",
		"provider=noop",
		"profile=driving",
		"status=ok",
		"source_point_count=3",
		"distance_meters=12.346",
		"duration_seconds=78.900",
		"geometry_format=geojson",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("summary missing %q in:\n%s", want, out)
		}
	}
}

func TestWriteRouteMatchGeoJSON(t *testing.T) {
	matchedAt := time.Date(2026, 5, 30, 1, 2, 3, 0, time.UTC)
	createdAt := time.Date(2026, 5, 30, 2, 3, 4, 0, time.UTC)
	run := storage.RouteMatchRun{
		ID:        9,
		CreatedAt: createdAt,
		Trace: routing.EnrichedTrace{
			Provider:        "noop",
			Profile:         "driving",
			Status:          "ok",
			SourcePointCount: 2,
			Geometry:        `{"type":"LineString","coordinates":[[151.0,-33.0],[151.1,-33.1]]}`,
			GeometryFormat:  "geojson",
			DistanceMeters:  100,
			DurationSeconds: 20,
			MatchedAt:       matchedAt,
		},
	}
	var buf bytes.Buffer
	if err := writeRouteMatchGeoJSON(&buf, run); err != nil {
		t.Fatalf("writeRouteMatchGeoJSON returned error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid GeoJSON output: %v", err)
	}
	if decoded["type"] != "Feature" {
		t.Fatalf("type = %v, want Feature", decoded["type"])
	}
	properties := decoded["properties"].(map[string]any)
	if properties["provider"] != "noop" {
		t.Fatalf("provider = %v, want noop", properties["provider"])
	}
}
