package ors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
		t.Fatalf("capabilities should report supported ORS operations: %+v", caps)
	}
}

func TestHealthUsesHealthEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2/health" {
			t.Fatalf("method=%s path=%q", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	if err := p.Health(context.Background()); err != nil {
		t.Fatalf("Health() err=%v", err)
	}
}

func TestRouteBuildsRequestParsesResultAndPreservesRaw(t *testing.T) {
	raw := `{"routes":[{"geometry":"abc","summary":{"distance":123.4,"duration":56.7}}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/directions/foot-walking/json" {
			t.Fatalf("method=%s path=%q", r.Method, r.URL.Path)
		}
		var body routeRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.GeometryFormat != "polyline5" {
			t.Fatalf("geometry_format=%q", body.GeometryFormat)
		}
		if len(body.Coordinates) != 2 || body.Coordinates[0][0] != 151.2 || body.Coordinates[0][1] != -33.8 {
			t.Fatalf("coordinates=%+v", body.Coordinates)
		}
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Route(context.Background(), routing.RouteRequest{
		Profile: "walking",
		Start:   routing.Coordinate{Lat: -33.8, Lng: 151.2},
		End:     routing.Coordinate{Lat: -33.9, Lng: 151.3},
	})
	if err != nil {
		t.Fatalf("Route() err=%v", err)
	}
	if result.Provider != Name || result.Profile != "foot-walking" || result.Status != "Ok" {
		t.Fatalf("unexpected identity/status: %+v", result)
	}
	if result.Geometry != "abc" || result.GeometryFormat != "polyline5" || result.DistanceMeters != 123.4 || result.DurationSeconds != 56.7 {
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

func TestMatchTraceUsesDirectionsRequest(t *testing.T) {
	raw := `{"routes":[{"geometry":"matched","summary":{"distance":200.5,"duration":42.25}}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/directions/driving-car/json" {
			t.Fatalf("method=%s path=%q", r.Method, r.URL.Path)
		}
		var body routeRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(body.Coordinates) != 2 || body.Coordinates[1][0] != 151.25 || body.Coordinates[1][1] != -33.85 {
			t.Fatalf("coordinates=%+v", body.Coordinates)
		}
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.MatchTrace(context.Background(), routing.MatchTraceRequest{Points: []routing.TracePoint{
		{Coordinate: routing.Coordinate{Lat: -33.8, Lng: 151.2}},
		{Coordinate: routing.Coordinate{Lat: -33.85, Lng: 151.25}},
	}})
	if err != nil {
		t.Fatalf("MatchTrace() err=%v", err)
	}
	if result.Geometry != "matched" || result.DistanceMeters != 200.5 || result.DurationSeconds != 42.25 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestMatchTraceRejectsTooFewPoints(t *testing.T) {
	p := NewWithConfig(Config{BaseURL: "http://example.invalid"})
	_, err := p.MatchTrace(context.Background(), routing.MatchTraceRequest{Points: []routing.TracePoint{{Coordinate: routing.Coordinate{Lat: -33.8, Lng: 151.2}}}})
	if err == nil || !strings.Contains(err.Error(), "at least two trace points") {
		t.Fatalf("MatchTrace() err=%v", err)
	}
}

func TestSnapBuildsRequestAndParsesPoints(t *testing.T) {
	raw := `{"locations":[{"location":[151.201,-33.801]},{"location":[151.301,-33.901]}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/snap/cycling-regular/json" {
			t.Fatalf("method=%s path=%q", r.Method, r.URL.Path)
		}
		var body snapRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(body.Locations) != 2 || body.Locations[0][0] != 151.2 || body.Locations[1][1] != -33.9 {
			t.Fatalf("locations=%+v", body.Locations)
		}
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Snap(context.Background(), routing.SnapRequest{Profile: "bike", Coordinates: []routing.Coordinate{{Lat: -33.8, Lng: 151.2}, {Lat: -33.9, Lng: 151.3}}})
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

func TestMatrixBuildsRequestAndParsesMatrices(t *testing.T) {
	raw := `{"durations":[[1],[3]],"distances":[[10],[30]]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/matrix/driving-car/json" {
			t.Fatalf("method=%s path=%q", r.Method, r.URL.Path)
		}
		var body matrixRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(body.Locations) != 3 || len(body.Sources) != 2 || body.Sources[1] != 1 || len(body.Destinations) != 1 || body.Destinations[0] != 2 {
			t.Fatalf("body=%+v", body)
		}
		if body.Units != "m" || len(body.Metrics) != 2 {
			t.Fatalf("metrics/units=%+v units=%q", body.Metrics, body.Units)
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
	raw := `{"durations":[[1,2]],"distances":[[10,20]]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(raw)) }))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	_, err := p.Matrix(context.Background(), routing.MatrixRequest{Sources: []routing.Coordinate{{Lat: -33.8, Lng: 151.2}}, Destinations: []routing.Coordinate{{Lat: -34, Lng: 151.4}}})
	if err == nil || !strings.Contains(err.Error(), "duration matrix row 0 has 2 columns, want 1") {
		t.Fatalf("Matrix() err=%v", err)
	}
}

func TestRouteProviderStatusErrorPreservesRawAndWarning(t *testing.T) {
	raw := `{"error":{"code":2010,"message":"Could not find routable point"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(raw)) }))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Route(context.Background(), routing.RouteRequest{Start: routing.Coordinate{Lat: -33.8, Lng: 151.2}, End: routing.Coordinate{Lat: -33.9, Lng: 151.3}})
	if err == nil {
		t.Fatal("Route() err=nil")
	}
	if !strings.Contains(err.Error(), "2010") {
		t.Fatalf("err=%v", err)
	}
	if result.Status != "2010" || string(result.RawResponse) != raw {
		t.Fatalf("result=%+v raw=%s", result, string(result.RawResponse))
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "Could not find routable point" {
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
		t.Fatalf("raw response=%q", string(result.RawResponse))
	}
}
