package valhalla

import (
	"context"
	"errors"
	"testing"

	"github.com/MickMake/GoTravel/routing"
)

func TestCapabilitiesConservative(t *testing.T) {
	p := New()
	caps := p.Capabilities(context.Background())
	if caps.Route || caps.MatchTrace || caps.Snap || caps.Matrix {
		t.Fatalf("capabilities should be conservative false values: %+v", caps)
	}
}

func TestNotImplementedOperations(t *testing.T) {
	p := New()
	ctx := context.Background()

	checks := []struct {
		name string
		err  error
	}{
		{name: "health", err: p.Health(ctx)},
		{name: "route", err: func() error { _, err := p.Route(ctx, routing.RouteRequest{}); return err }()},
		{name: "match", err: func() error { _, err := p.MatchTrace(ctx, routing.MatchTraceRequest{}); return err }()},
		{name: "snap", err: func() error { _, err := p.Snap(ctx, routing.SnapRequest{}); return err }()},
		{name: "matrix", err: func() error { _, err := p.Matrix(ctx, routing.MatrixRequest{}); return err }()},
	}

	for _, c := range checks {
		if !errors.Is(c.err, routing.ErrNotImplemented) {
			t.Fatalf("%s err=%v want ErrNotImplemented", c.name, c.err)
		}
	}
}
