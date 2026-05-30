package noop

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/MickMake/GoTravel/routing"
)

func TestCapabilitiesConservative(t *testing.T) {
	p := New()
	caps := p.Capabilities(context.Background())
	if caps.Route || !caps.MatchTrace || caps.Snap || caps.Matrix {
		t.Fatalf("capabilities should only advertise trace matching: %+v", caps)
	}
}

func TestMatchTrace(t *testing.T) {
	p := New()
	result, err := p.MatchTrace(context.Background(), routing.MatchTraceRequest{
		Profile: "driving",
		Points: []routing.TracePoint{
			{Coordinate: routing.Coordinate{Lat: -33.8, Lng: 151.2}, Time: time.Unix(1700000000, 0)},
			{Coordinate: routing.Coordinate{Lat: -33.9, Lng: 151.3}, Time: time.Unix(1700000060, 0)},
		},
	})
	if err != nil {
		t.Fatalf("MatchTrace returned error: %v", err)
	}
	if result.Provider != Name || result.Profile != "driving" || result.Status != "ok" || result.GeometryFormat != "geojson" {
		t.Fatalf("unexpected result metadata: %+v", result)
	}
	var geometry map[string]any
	if err := json.Unmarshal([]byte(result.Geometry), &geometry); err != nil {
		t.Fatalf("geometry is not JSON: %v", err)
	}
	if geometry["type"] != "LineString" {
		t.Fatalf("geometry type = %v, want LineString", geometry["type"])
	}
}

func TestNotImplementedOperations(t *testing.T) {
	p := New()
	ctx := context.Background()

	checks := []struct {
		name string
		err  error
	}{
		{name: "route", err: func() error { _, err := p.Route(ctx, routing.RouteRequest{}); return err }()},
		{name: "snap", err: func() error { _, err := p.Snap(ctx, routing.SnapRequest{}); return err }()},
		{name: "matrix", err: func() error { _, err := p.Matrix(ctx, routing.MatrixRequest{}); return err }()},
	}

	for _, c := range checks {
		if !errors.Is(c.err, routing.ErrNotImplemented) {
			t.Fatalf("%s err=%v want ErrNotImplemented", c.name, c.err)
		}
	}
}
