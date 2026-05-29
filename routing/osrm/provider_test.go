package osrm

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

func TestCapabilitiesSupported(t *testing.T) {
	p := New()
	caps := p.Capabilities(context.Background())
	if !caps.Route || !caps.MatchTrace || !caps.Snap || !caps.Matrix {
		t.Fatalf("capabilities should report supported OSRM operations: %+v", caps)
	}
}

func TestHealthUsesNearestAndTreatsReachableOSRMAsHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nearest/v1/driving/0,0" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		if got := r.URL.Query().Get("number"); got != "1" {
			t.Fatalf("number=%q", got)
		}
		_, _ = w.Write([]byte(`{"code":"NoSegment","message":"no road nearby"}`))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	if err := p.Health(context.Background()); err != nil {
		t.Fatalf("Health() err=%v", err)
	}
}

func TestRouteBuildsRequestParsesResultAndPreservesRaw(t *testing.T) {
	raw := `{"code":"Ok","routes":[{"geometry":"abc","distance":123.4,"duration":56.7}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/route/v1/walking/151.2,-33.8;151.3,-33.9" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		if got := r.URL.Query().Get("overview"); got != "full" {
			t.Fatalf("overview=%q", got)
		}
		if got := r.URL.Query().Get("geometries"); got != "polyline6" {
			t.Fatalf("geometries=%q", got)
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
	if result.Provider != Name || result.Profile != "walking" || result.Status != "Ok" {
		t.Fatalf("unexpected identity/status: %+v", result)
	}
	if result.Geometry != "abc" || result.GeometryFormat != "polyline6" || result.DistanceMeters != 123.4 || result.DurationSeconds != 56.7 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if string(result.RawResponse) != raw {
		t.Fatalf("raw response not preserved: %s", string(result.RawResponse))
	}
}

func TestMatchTraceBuildsTimestampsAndRadiuses(t *testing.T) {
	confidence := 0.87
	raw := `{"code":"Ok","matchings":[{"geometry":"matched","distance":200.5,"duration":42.25,"confidence":0.87}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/match/v1/driving/151.2,-33.8;151.25,-33.85" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		if got := r.URL.Query().Get("timestamps"); got != "1700000000;1700000060" {
			t.Fatalf("timestamps=%q", got)
		}
		if got := r.URL.Query().Get("radiuses"); got != "5;unlimited" {
			t.Fatalf("radiuses=%q", got)
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
	if result.Geometry != "matched" || result.DistanceMeters != 200.5 || result.DurationSeconds != 42.25 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Confidence == nil || *result.Confidence != confidence {
		t.Fatalf("confidence=%v", result.Confidence)
	}
}

func TestSnapSupportsMultipleCoordinatesWithNearestRequests(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.URL.Query().Get("number"); got != "1" {
			t.Fatalf("number=%q", got)
		}
		switch r.URL.Path {
		case "/nearest/v1/driving/151.2,-33.8":
			_, _ = w.Write([]byte(`{"code":"Ok","waypoints":[{"location":[151.201,-33.801]}]}`))
		case "/nearest/v1/driving/151.3,-33.9":
			_, _ = w.Write([]byte(`{"code":"Ok","waypoints":[{"location":[151.301,-33.901]}]}`))
		default:
			t.Fatalf("path=%q", r.URL.Path)
		}
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Snap(context.Background(), routing.SnapRequest{Coordinates: []routing.Coordinate{
		{Lat: -33.8, Lng: 151.2},
		{Lat: -33.9, Lng: 151.3},
	}})
	if err != nil {
		t.Fatalf("Snap() err=%v", err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
	if len(result.Points) != 2 {
		t.Fatalf("points=%+v", result.Points)
	}
	if result.Points[0].Lat != -33.801 || result.Points[0].Lng != 151.201 || result.Points[1].Lat != -33.901 || result.Points[1].Lng != 151.301 {
		t.Fatalf("unexpected snapped points: %+v", result.Points)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(result.RawResponse, &raw); err != nil {
		t.Fatalf("raw response should be aggregate JSON array: %v", err)
	}
	if len(raw) != 2 {
		t.Fatalf("raw response count=%d", len(raw))
	}
}

func TestMatrixBuildsTableRequestAndParsesMatrices(t *testing.T) {
	raw := `{"code":"Ok","durations":[[1,2],[3,4]],"distances":[[10,20],[30,40]]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/table/v1/driving/151.2,-33.8;151.3,-33.9;151.4,-34" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		if got := r.URL.Query().Get("sources"); got != "0;1" {
			t.Fatalf("sources=%q", got)
		}
		if got := r.URL.Query().Get("destinations"); got != "2" {
			t.Fatalf("destinations=%q", got)
		}
		if got := r.URL.Query().Get("annotations"); got != "duration,distance" {
			t.Fatalf("annotations=%q", got)
		}
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Matrix(context.Background(), routing.MatrixRequest{
		Sources: []routing.Coordinate{
			{Lat: -33.8, Lng: 151.2},
			{Lat: -33.9, Lng: 151.3},
		},
		Destinations: []routing.Coordinate{{Lat: -34, Lng: 151.4}},
	})
	if err != nil {
		t.Fatalf("Matrix() err=%v", err)
	}
	if result.DurationMatrix[1][0] != 3 || result.DistanceMatrix[1][1] != 40 {
		t.Fatalf("unexpected matrices: %+v %+v", result.DurationMatrix, result.DistanceMatrix)
	}
	if string(result.RawResponse) != raw {
		t.Fatalf("raw response not preserved: %s", string(result.RawResponse))
	}
}

func TestRouteProviderStatusErrorPreservesRawAndWarning(t *testing.T) {
	raw := `{"code":"InvalidUrl","message":"broken coordinates"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL})
	result, err := p.Route(context.Background(), routing.RouteRequest{})
	if err == nil {
		t.Fatal("Route() err=nil")
	}
	if !strings.Contains(err.Error(), "InvalidUrl") {
		t.Fatalf("err=%v", err)
	}
	if result.Status != "InvalidUrl" || string(result.RawResponse) != raw {
		t.Fatalf("result=%+v raw=%s", result, string(result.RawResponse))
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "broken coordinates" {
		t.Fatalf("warnings=%+v", result.Warnings)
	}
}
