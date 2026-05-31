package ors

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/MickMake/GoTravel/routing"
)

func TestNameAndDefaults(t *testing.T) {
	p := New()
	if p.Name() != Name {
		t.Fatalf("Name()=%q want %q", p.Name(), Name)
	}
	if p.baseURL != defaultBaseURL {
		t.Fatalf("baseURL=%q want %q", p.baseURL, defaultBaseURL)
	}
	if p.profile != defaultProfile {
		t.Fatalf("profile=%q want %q", p.profile, defaultProfile)
	}
}

func TestNewWithConfigUsesConfig(t *testing.T) {
	client := &http.Client{}
	p := NewWithConfig(Config{BaseURL: "http://example.test/ors/", Profile: "cycling-regular", HTTPClient: client})
	if p.baseURL != "http://example.test/ors" {
		t.Fatalf("baseURL=%q", p.baseURL)
	}
	if p.profile != "cycling-regular" {
		t.Fatalf("profile=%q", p.profile)
	}
	if p.httpClient != client {
		t.Fatalf("HTTPClient was not preserved")
	}
}

func TestProfileAliasesMapToORSProfiles(t *testing.T) {
	tests := map[string]string{
		"driving": "driving-car",
		"car":     "driving-car",
		"walking": "foot-walking",
		"foot":    "foot-walking",
		"cycling": "cycling-regular",
		"bike":    "cycling-regular",
	}
	p := New()
	for alias, want := range tests {
		t.Run(alias, func(t *testing.T) {
			result, err := p.MatchTrace(context.Background(), routing.MatchTraceRequest{Profile: alias})
			if !errors.Is(err, routing.ErrNotImplemented) {
				t.Fatalf("MatchTrace() err=%v want ErrNotImplemented", err)
			}
			if result.Profile != want {
				t.Fatalf("profile=%q want %q", result.Profile, want)
			}
		})
	}
}

func TestORSNativeProfilesPassThrough(t *testing.T) {
	nativeProfiles := []string{"driving-car", "foot-walking", "cycling-regular", "wheelchair"}
	p := New()
	for _, profile := range nativeProfiles {
		t.Run(profile, func(t *testing.T) {
			result, err := p.MatchTrace(context.Background(), routing.MatchTraceRequest{Profile: profile})
			if !errors.Is(err, routing.ErrNotImplemented) {
				t.Fatalf("MatchTrace() err=%v want ErrNotImplemented", err)
			}
			if result.Profile != profile {
				t.Fatalf("profile=%q want %q", result.Profile, profile)
			}
		})
	}
}

func TestConfiguredProfileAliasMapsToORSProfile(t *testing.T) {
	result, err := NewWithConfig(Config{Profile: "walking"}).MatchTrace(context.Background(), routing.MatchTraceRequest{})
	if !errors.Is(err, routing.ErrNotImplemented) {
		t.Fatalf("MatchTrace() err=%v want ErrNotImplemented", err)
	}
	if result.Profile != "foot-walking" {
		t.Fatalf("profile=%q want foot-walking", result.Profile)
	}
}

func TestCapabilitiesAreHonest(t *testing.T) {
	caps := New().Capabilities(context.Background())
	want := routing.Capabilities{Route: true, MatchTrace: false, Snap: false, Matrix: true}
	if caps != want {
		t.Fatalf("capabilities=%+v want %+v", caps, want)
	}
}

func TestHealthRemainsUnimplemented(t *testing.T) {
	if err := New().Health(context.Background()); !errors.Is(err, routing.ErrNotImplemented) {
		t.Fatalf("Health() err=%v want ErrNotImplemented", err)
	}
}

func TestRouteBuildsDirectionsGeoJSONRequestParsesResultAndPreservesRaw(t *testing.T) {
	raw := `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{"summary":{"distance":123.4,"duration":56.7}},"geometry":{"type":"LineString","coordinates":[[151.2,-33.8],[151.3,-33.9]]}}],"metadata":{"query":{"warning":"small warning"}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%q", r.Method)
		}
		if r.URL.Path != "/ors/v2/directions/foot-walking/geojson" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		var body struct {
			Coordinates [][]float64 `json:"coordinates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		want := [][]float64{{151.2, -33.8}, {151.3, -33.9}}
		if !reflect.DeepEqual(body.Coordinates, want) {
			t.Fatalf("coordinates=%v want %v", body.Coordinates, want)
		}
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	p := NewWithConfig(Config{BaseURL: server.URL + "/ors"})
	result, err := p.Route(context.Background(), routing.RouteRequest{
		Profile: "foot-walking",
		Start:   routing.Coordinate{Lat: -33.8, Lng: 151.2},
		End:     routing.Coordinate{Lat: -33.9, Lng: 151.3},
	})
	if err != nil {
		t.Fatalf("Route() err=%v", err)
	}
	if result.Provider != Name || result.Profile != "foot-walking" || result.Status != "Ok" {
		t.Fatalf("unexpected identity/status: %+v", result)
	}
	if result.GeometryFormat != "geojson" || !strings.Contains(result.Geometry, `"LineString"`) {
		t.Fatalf("unexpected geometry: format=%q geometry=%s", result.GeometryFormat, result.Geometry)
	}
	if result.DistanceMeters != 123.4 || result.DurationSeconds != 56.7 {
		t.Fatalf("unexpected summary: %+v", result)
	}
	if string(result.RawResponse) != raw {
		t.Fatalf("raw response not preserved: %s", string(result.RawResponse))
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "small warning" {
		t.Fatalf("warnings=%+v", result.Warnings)
	}
}

func TestRouteRejectsSameStartAndEnd(t *testing.T) {
	_, err := NewWithConfig(Config{BaseURL: "http://example.invalid"}).Route(context.Background(), routing.RouteRequest{
		Start: routing.Coordinate{Lat: -33.8, Lng: 151.2},
		End:   routing.Coordinate{Lat: -33.8, Lng: 151.2},
	})
	if err == nil || !strings.Contains(err.Error(), "start and end coordinates must differ") {
		t.Fatalf("Route() err=%v", err)
	}
}

func TestMatrixBuildsRequestParsesMatricesAndPreservesRaw(t *testing.T) {
	raw := `{"durations":[[1],[3]],"distances":[[10],[30]]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%q", r.Method)
		}
		if r.URL.Path != "/v2/matrix/driving-car" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		var body struct {
			Locations    [][]float64 `json:"locations"`
			Sources      []int       `json:"sources"`
			Destinations []int       `json:"destinations"`
			Metrics      []string    `json:"metrics"`
			Units        string      `json:"units"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !reflect.DeepEqual(body.Locations, [][]float64{{151.2, -33.8}, {151.3, -33.9}, {151.4, -34}}) {
			t.Fatalf("locations=%v", body.Locations)
		}
		if !reflect.DeepEqual(body.Sources, []int{0, 1}) || !reflect.DeepEqual(body.Destinations, []int{2}) {
			t.Fatalf("sources=%v destinations=%v", body.Sources, body.Destinations)
		}
		if !reflect.DeepEqual(body.Metrics, []string{"duration", "distance"}) || body.Units != "m" {
			t.Fatalf("metrics=%v units=%q", body.Metrics, body.Units)
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
	if result.Status != "Ok" || result.DurationMatrix[1][0] != 3 || result.DistanceMatrix[1][0] != 30 {
		t.Fatalf("unexpected matrix result: %+v", result)
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
	raw := `{"durations":[[1,2],[3,4]],"distances":[[10,20],[30,40]]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	_, err := NewWithConfig(Config{BaseURL: server.URL}).Matrix(context.Background(), routing.MatrixRequest{
		Sources: []routing.Coordinate{
			{Lat: -33.8, Lng: 151.2},
			{Lat: -33.9, Lng: 151.3},
		},
		Destinations: []routing.Coordinate{{Lat: -34, Lng: 151.4}},
	})
	if err == nil || !strings.Contains(err.Error(), "duration matrix row 0 has 2 columns, want 1") {
		t.Fatalf("Matrix() err=%v", err)
	}
}

func TestSnapReturnsNotImplemented(t *testing.T) {
	result, err := New().Snap(context.Background(), routing.SnapRequest{Coordinates: []routing.Coordinate{{Lat: -33.8, Lng: 151.2}}})
	if !errors.Is(err, routing.ErrNotImplemented) {
		t.Fatalf("Snap() err=%v want ErrNotImplemented", err)
	}
	if result.Provider != Name || result.Profile != defaultProfile {
		t.Fatalf("unexpected result metadata: %+v", result)
	}
}

func TestMatchTraceReturnsNotImplemented(t *testing.T) {
	result, err := New().MatchTrace(context.Background(), routing.MatchTraceRequest{})
	if !errors.Is(err, routing.ErrNotImplemented) {
		t.Fatalf("MatchTrace() err=%v want ErrNotImplemented", err)
	}
	if result.Provider != Name || result.Profile != defaultProfile {
		t.Fatalf("unexpected result metadata: %+v", result)
	}
}

func TestProviderStatusErrorPreservesRawAndWarning(t *testing.T) {
	raw := `{"error":{"code":2010,"message":"bad request"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	result, err := NewWithConfig(Config{BaseURL: server.URL}).Route(context.Background(), routing.RouteRequest{
		Start: routing.Coordinate{Lat: -33.8, Lng: 151.2},
		End:   routing.Coordinate{Lat: -33.9, Lng: 151.3},
	})
	if err == nil || !strings.Contains(err.Error(), "2010") {
		t.Fatalf("Route() err=%v", err)
	}
	if result.Status != "2010" || string(result.RawResponse) != raw {
		t.Fatalf("result=%+v raw=%s", result, string(result.RawResponse))
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "bad request" {
		t.Fatalf("warnings=%+v", result.Warnings)
	}
}

func TestHTTPErrorPreservesRaw(t *testing.T) {
	raw := `plain HTTP failure`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, raw, http.StatusBadRequest)
	}))
	defer server.Close()

	result, err := NewWithConfig(Config{BaseURL: server.URL}).Matrix(context.Background(), routing.MatrixRequest{
		Sources:      []routing.Coordinate{{Lat: -33.8, Lng: 151.2}},
		Destinations: []routing.Coordinate{{Lat: -33.9, Lng: 151.3}},
	})
	if err == nil || !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("Matrix() err=%v", err)
	}
	if !strings.Contains(string(result.RawResponse), raw) {
		t.Fatalf("raw response not preserved: %s", string(result.RawResponse))
	}
}
