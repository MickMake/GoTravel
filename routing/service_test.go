package routing

import (
	"context"
	"errors"
	"testing"
)

func TestNewServiceRejectsNilProvider(t *testing.T) {
	service, err := NewService(nil)
	if service != nil {
		t.Fatalf("service=%v want nil", service)
	}
	if !errors.Is(err, ErrNilProvider) {
		t.Fatalf("err=%v want ErrNilProvider", err)
	}
}

func TestServiceDelegatesToProvider(t *testing.T) {
	provider := &fakeProvider{name: "fake"}
	service, err := NewService(provider)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}
	ctx := context.Background()

	if got := service.Provider(); got != provider {
		t.Fatalf("Provider()=%v want fake provider", got)
	}
	if err := service.Health(ctx); err != nil {
		t.Fatalf("Health() err=%v", err)
	}
	if caps := service.Capabilities(ctx); !caps.Route || !caps.MatchTrace || !caps.Snap || !caps.Matrix {
		t.Fatalf("Capabilities()=%+v", caps)
	}

	route, err := service.Route(ctx, RouteRequest{Profile: "driving"})
	if err != nil {
		t.Fatalf("Route() err=%v", err)
	}
	if route.Provider != "fake" || route.Profile != "driving" {
		t.Fatalf("Route()=%+v", route)
	}

	match, err := service.MatchTrace(ctx, MatchTraceRequest{Profile: "walking"})
	if err != nil {
		t.Fatalf("MatchTrace() err=%v", err)
	}
	if match.Provider != "fake" || match.Profile != "walking" {
		t.Fatalf("MatchTrace()=%+v", match)
	}

	snap, err := service.Snap(ctx, SnapRequest{Profile: "cycling"})
	if err != nil {
		t.Fatalf("Snap() err=%v", err)
	}
	if snap.Provider != "fake" || snap.Profile != "cycling" {
		t.Fatalf("Snap()=%+v", snap)
	}

	matrix, err := service.Matrix(ctx, MatrixRequest{Profile: "truck"})
	if err != nil {
		t.Fatalf("Matrix() err=%v", err)
	}
	if matrix.Provider != "fake" || matrix.Profile != "truck" {
		t.Fatalf("Matrix()=%+v", matrix)
	}
}

type fakeProvider struct {
	name string
}

func (p *fakeProvider) Name() string { return p.name }
func (p *fakeProvider) Health(ctx context.Context) error { return nil }
func (p *fakeProvider) Capabilities(ctx context.Context) Capabilities {
	return Capabilities{Route: true, MatchTrace: true, Snap: true, Matrix: true}
}
func (p *fakeProvider) Route(ctx context.Context, req RouteRequest) (RouteResult, error) {
	return RouteResult{Provider: p.name, Profile: req.Profile}, nil
}
func (p *fakeProvider) MatchTrace(ctx context.Context, req MatchTraceRequest) (MatchTraceResult, error) {
	return MatchTraceResult{Provider: p.name, Profile: req.Profile}, nil
}
func (p *fakeProvider) Snap(ctx context.Context, req SnapRequest) (SnapResult, error) {
	return SnapResult{Provider: p.name, Profile: req.Profile}, nil
}
func (p *fakeProvider) Matrix(ctx context.Context, req MatrixRequest) (MatrixResult, error) {
	return MatrixResult{Provider: p.name, Profile: req.Profile}, nil
}
