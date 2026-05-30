package valhalla

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MickMake/GoTravel/routing"
)

func TestNameCapabilitiesAndDefaults(t *testing.T) {
	p := New()
	if p.Name() != Name {
		t.Fatalf("Name()=%q", p.Name())
	}
	if p.baseURL != defaultBaseURL || p.profile != defaultProfile {
		t.Fatalf("defaults baseURL=%q profile=%q", p.baseURL, p.profile)
	}
	caps := p.Capabilities(context.Background())
	if !caps.Route || !caps.MatchTrace || !caps.Snap || !caps.Matrix {
		t.Fatalf("capabilities should report supported Valhalla operations: %+v", caps)
	}
}

func TestHealthUsesLocate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/locate" {
			t.Fatalf("method=%s path=%q", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["costing"] != "auto" {
			t.Fatalf("costing=%v", body["costing"])
		}
		_, _ = w.Write([]byte(`[{"nodes":[{"lat":1,"lon":2}]}]`))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	if err := p.Health(context.Background()); err != nil {
		t.Fatalf("Health() err=%v", err)
	}
}

func TestRouteBuildsRequestParsesResultAndPreservesRaw(t *testing.T) {
	raw := `{"trip":{"status":0,"status_message":"Found route","summary":{"length":1.234,"time":56.7},"legs":[{"shape":"abc"}]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/route" {
			t.Fatalf("method=%s path=%q", r.Method, r.URL.Path)
		}
		var body routeRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Costing != "pedestrian" || body.ShapeFormat != "polyline6" {
			t.Fatalf("body=%+v", body)
		}
		if len(body.Locations) != 2 || body.Locations[0].Lat != -33.8 || body.Locations[0].Lon != 151.2 {
			t.Fatalf("locations=%+v", body.Locations)
		}
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Route(context.Background(), routing.RouteRequest{
		Profile: "pedestrian",
		Start:   routing.Coordinate{Lat: -33.8, Lng: 151.2},
		End:     routing.Coordinate{Lat: -33.9, Lng: 151.3},
	})
	if err != nil {
		t.Fatalf("Route() err=%v", err)
	}
	if result.Provider != Name || result.Profile != "pedestrian" || result.Status != "Ok" {
		t.Fatalf("unexpected identity/status: %+v", result)
	}
	if result.Geometry != "abc" || result.GeometryFormat != "polyline6" || result.DistanceMeters != 1234 || result.DurationSeconds != 56.7 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if string(result.RawResponse) != raw {
		t.Fatalf("raw response not preserved: %s", string(result.RawResponse))
	}
}

func TestRouteRejectsSameStartAndEnd(t *testing.T) {
	p := NewWithConfig(Config{BaseURL: "http://example.invalid"})
	_, err := p.Route(context.Background(), routing.RouteRequest{Start: routing.Coordinate{Lat: -33.8, Lng: 151.2}, End: routing.Coordinate{Lat: -33.8, Lng: 151.2}})
	if err == nil || !strings.Contains(err.Error(), "start and end coordinates must differ") {
		t.Fatalf("Route() err=%v", err)
	}
}

func TestMatchTraceBuildsTraceRouteRequest(t *testing.T) {
	confidence := 0.92
	raw := `{"trip":{"status":0,"summary":{"length":0.2,"time":42.25},"confidence":0.92,"legs":[{"shape":"matched"}]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/trace_route" {
			t.Fatalf("method=%s path=%q", r.Method, r.URL.Path)
		}
		var body traceRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Costing != "auto" || body.ShapeMatch != "map_snap" || body.ShapeFormat != "polyline6" {
			t.Fatalf("body=%+v", body)
		}
		if len(body.Shape) != 2 || body.Shape[0].Time != 1700000000 || body.Shape[0].SearchRadius == nil || *body.Shape[0].SearchRadius != 5 {
			t.Fatalf("shape=%+v", body.Shape)
		}
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	radius := 5.0
	result, err := p.MatchTrace(context.Background(), routing.MatchTraceRequest{Points: []routing.TracePoint{
		{Coordinate: routing.Coordinate{Lat: -33.8, Lng: 151.2}, Time: time.Unix(1700000000, 0), Radius: &radius},
		{Coordinate: routing.Coordinate{Lat: -33.85, Lng: 151.25}, Time: time.Unix(1700000060, 0)},
	}})
	if err != nil {
		t.Fatalf("MatchTrace() err=%v", err)
	}
	if result.Geometry != "matched" || result.DistanceMeters != 200 || result.DurationSeconds != 42.25 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Confidence == nil || *result.Confidence != confidence {
		t.Fatalf("confidence=%v", result.Confidence)
	}
}

func TestMatchTraceRejectsTooFewPoints(t *testing.T) {
	p := NewWithConfig(Config{BaseURL: "http://example.invalid"})
	_, err := p.MatchTrace(context.Background(), routing.MatchTraceRequest{Points: []routing.TracePoint{{Coordinate: routing.Coordinate{Lat: -33.8, Lng: 151.2}}}})
	if err == nil || !strings.Contains(err.Error(), "at least two trace points") {
		t.Fatalf("MatchTrace() err=%v", err)
	}
}

func TestSnapBuildsLocateRequestAndParsesPoints(t *testing.T) {
	raw := `[{"nodes":[{"lat":-33.801,"lon":151.201}]},{"nodes":[{"lat":-33.901,"lon":151.301}]}]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/locate" {
			t.Fatalf("method=%s path=%q", r.Method, r.URL.Path)
		}
		var body locateRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Costing != "bicycle" || len(body.Locations) != 2 {
			t.Fatalf("body=%+v", body)
		}
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Snap(context.Background(), routing.SnapRequest{Profile: "bicycle", Coordinates: []routing.Coordinate{{Lat: -33.8, Lng: 151.2}, {Lat: -33.9, Lng: 151.3}}})
	if err != nil {
		t.Fatalf("Snap() err=%v", err)
	}
	if len(result.Points) != 2 || result.Points[0].Lat != -33.801 || result.Points[0].Lng != 151.201 || result.Points[1].Lat != -33.901 || result.Points[1].Lng != 151.301 {
		t.Fatalf("unexpected snapped points: %+v", result.Points)
	}
	if string(result.RawResponse) != raw {
		t.Fatalf("raw response not preserved: %s", string(result.RawResponse))
	}
}

func TestMatrixBuildsSourcesToTargetsRequestAndParsesMatrices(t *testing.T) {
	raw := `{"sources_to_targets":[[{"distance":0.01,"time":1}],[{"distance":0.03,"time":3}]]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/sources_to_targets" {
			t.Fatalf("method=%s path=%q", r.Method, r.URL.Path)
		}
		var body matrixRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Costing != "auto" || body.Units != "kilometers" || len(body.Sources) != 2 || len(body.Targets) != 1 {
			t.Fatalf("body=%+v", body)
		}
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Matrix(context.Background(), routing.MatrixRequest{Sources: []routing.Coordinate{{Lat: -33.8, Lng: 151.2}, {Lat: -33.9, Lng: 151.3}}, Destinations: []routing.Coordinate{{Lat: -34, Lng: 151.4}}})
	if err != nil {
		t.Fatalf("Matrix() err=%v", err)
	}
	if result.DurationMatrix[1][0] != 3 || result.DistanceMatrix[1][0] != 30 {
		t.Fatalf("unexpected matrices: %+v %+v", result.DurationMatrix, result.DistanceMatrix)
	}
	if string(result.RawResponse) != raw {
		t.Fatalf("raw response not preserved: %s", string(result.RawResponse))
	}
}

func TestMatrixRejectsEmptySourcesOrDestinations(t *testing.T) {
	p := NewWithConfig(Config{BaseURL: "http://example.invalid"})
	_, err := p.Matrix(context.Background(), routing.MatrixRequest{Destinations: []routing.Coordinate{{Lat: -34, Lng: 151.4}}})
	if err == nil || !strings.Contains(err.Error(), "at least one source") {
		t.Fatalf("Matrix() missing source err=%v", err)
	}
	_, err = p.Matrix(context.Background(), routing.MatrixRequest{Sources: []routing.Coordinate{{Lat: -33.8, Lng: 151.2}}})
	if err == nil || !strings.Contains(err.Error(), "at least one destination") {
		t.Fatalf("Matrix() missing destination err=%v", err)
	}
}

func TestMatrixRejectsUnexpectedDimensions(t *testing.T) {
	raw := `{"sources_to_targets":[[{"distance":0.01,"time":1},{"distance":0.02,"time":2}]]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(raw)) }))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	_, err := p.Matrix(context.Background(), routing.MatrixRequest{Sources: []routing.Coordinate{{Lat: -33.8, Lng: 151.2}}, Destinations: []routing.Coordinate{{Lat: -34, Lng: 151.4}}})
	if err == nil || !strings.Contains(err.Error(), "matrix row 0 has 2 columns, want 1") {
		t.Fatalf("Matrix() err=%v", err)
	}
}

func TestRouteProviderStatusErrorPreservesRawAndWarning(t *testing.T) {
	raw := `{"trip":{"status":171,"status_message":"No path could be found"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(raw)) }))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Route(context.Background(), routing.RouteRequest{Start: routing.Coordinate{Lat: -33.8, Lng: 151.2}, End: routing.Coordinate{Lat: -33.9, Lng: 151.3}})
	if err == nil {
		t.Fatal("Route() err=nil")
	}
	if !strings.Contains(err.Error(), "171") {
		t.Fatalf("err=%v", err)
	}
	if result.Status != "171" || string(result.RawResponse) != raw {
		t.Fatalf("result=%+v raw=%s", result, string(result.RawResponse))
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "No path could be found" {
		t.Fatalf("warnings=%+v", result.Warnings)
	}
}

func TestHTTPErrorPreservesRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "not ready", http.StatusServiceUnavailable) }))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Route(context.Background(), routing.RouteRequest{Start: routing.Coordinate{Lat: -33.8, Lng: 151.2}, End: routing.Coordinate{Lat: -33.9, Lng: 151.3}})
	if err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("Route() err=%v", err)
	}
	if !strings.Contains(string(result.RawResponse), "not ready") {
		t.Fatalf("raw=%q", string(result.RawResponse))
	}
}

type routeRequestBody struct {
	Costing     string             `json:"costing"`
	ShapeFormat string             `json:"shape_format"`
	Locations   []valhallaLocation `json:"locations"`
}

type traceRequestBody struct {
	Costing     string               `json:"costing"`
	ShapeFormat string               `json:"shape_format"`
	ShapeMatch  string               `json:"shape_match"`
	Shape       []valhallaTracePoint `json:"shape"`
}

type locateRequestBody struct {
	Costing   string             `json:"costing"`
	Locations []valhallaLocation `json:"locations"`
}

type matrixRequestBody struct {
	Costing string             `json:"costing"`
	Sources []valhallaLocation `json:"sources"`
	Targets []valhallaLocation `json:"targets"`
	Units   string             `json:"units"`
}
