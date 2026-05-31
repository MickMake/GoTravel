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
			Provider:         "noop",
			Profile:          "driving",
			Status:           "ok",
			SourcePointCount: 3,
			DistanceMeters:   12.3456,
			DurationSeconds:  78.9,
			GeometryFormat:   "geojson",
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
			Provider:         "noop",
			Profile:          "driving",
			Status:           "ok",
			SourcePointCount: 2,
			Geometry:         `{"type":"LineString","coordinates":[[151.0,-33.0],[151.1,-33.1]]}`,
			GeometryFormat:   "geojson",
			DistanceMeters:   100,
			DurationSeconds:  20,
			MatchedAt:        matchedAt,
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

func TestWriteRouteMatchGeoJSONConvertsPolyline(t *testing.T) {
	run := storage.RouteMatchRun{
		ID: 10,
		Trace: routing.EnrichedTrace{
			Provider:       "osrm",
			Profile:        "driving",
			Status:         "ok",
			Geometry:       "_p~iF~ps|U_ulLnnqC_mqNvxq`@",
			GeometryFormat: "polyline",
		},
	}
	var buf bytes.Buffer
	if err := writeRouteMatchGeoJSON(&buf, run); err != nil {
		t.Fatalf("writeRouteMatchGeoJSON returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"type": "LineString"`) || !strings.Contains(out, "-120.2") {
		t.Fatalf("GeoJSON output did not include decoded LineString:\n%s", out)
	}
}

func TestRenderRouteMatchExportHandlesCombinedGeoJSONGeometry(t *testing.T) {
	run := storage.RouteMatchRun{
		ID: 13,
		Trace: routing.EnrichedTrace{
			Provider:         "osrm",
			Profile:          "driving",
			Status:           "Ok",
			SourcePointCount: 101,
			Geometry:         `{"type":"LineString","coordinates":[[151.0,-33.0],[151.1,-33.1],[151.2,-33.2]]}`,
			GeometryFormat:   "geojson",
		},
	}

	geojson, err := renderRouteMatchExport("geojson", run)
	if err != nil {
		t.Fatalf("renderRouteMatchExport geojson returned error: %v", err)
	}
	if !strings.Contains(string(geojson), `"source_point_count": 101`) || !strings.Contains(string(geojson), `151.2`) {
		t.Fatalf("GeoJSON export missing combined geometry details:\n%s", string(geojson))
	}

	gpx, err := renderRouteMatchExport("gpx", run)
	if err != nil {
		t.Fatalf("renderRouteMatchExport gpx returned error: %v", err)
	}
	if !strings.Contains(string(gpx), `<trkpt lat="-33.2000000" lon="151.2000000"></trkpt>`) {
		t.Fatalf("GPX export missing combined geometry point:\n%s", string(gpx))
	}
}

func TestRenderRouteMatchExportRejectsUnsupportedGeometry(t *testing.T) {
	run := storage.RouteMatchRun{
		ID: 12,
		Trace: routing.EnrichedTrace{
			Geometry:       "abc",
			GeometryFormat: "unsupported",
		},
	}
	_, err := renderRouteMatchExport("geojson", run)
	if err == nil || !strings.Contains(err.Error(), "unsupported route geometry format") {
		t.Fatalf("err=%v, want unsupported geometry format", err)
	}
}

func TestWriteRouteMatchGPX(t *testing.T) {
	run := storage.RouteMatchRun{
		ID: 11,
		Trace: routing.EnrichedTrace{
			Provider:       "osrm",
			Profile:        "driving",
			Status:         "ok",
			Geometry:       `{"type":"LineString","coordinates":[[151.0,-33.0],[151.1,-33.1]]}`,
			GeometryFormat: "geojson",
		},
	}
	var buf bytes.Buffer
	if err := writeRouteMatchGPX(&buf, run); err != nil {
		t.Fatalf("writeRouteMatchGPX returned error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		`<gpx version="1.1" creator="GoTravel"`,
		`<name>GoTravel route match 11</name>`,
		`<trkpt lat="-33.0000000" lon="151.0000000"></trkpt>`,
		`<trkpt lat="-33.1000000" lon="151.1000000"></trkpt>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("GPX output missing %q in:\n%s", want, out)
		}
	}
}
