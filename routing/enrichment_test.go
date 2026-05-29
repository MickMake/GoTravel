package routing

import (
	"context"
	"errors"
	"testing"
)

func TestNewEnricherRejectsNilService(t *testing.T) {
	enricher, err := NewEnricher(nil)
	if enricher != nil {
		t.Fatalf("enricher=%v want nil", enricher)
	}
	if !errors.Is(err, ErrNilService) {
		t.Fatalf("err=%v want ErrNilService", err)
	}
}

func TestEnricherMatchTraceDelegatesToService(t *testing.T) {
	provider := &fakeEnrichmentProvider{name: "fake"}
	service, err := NewService(provider)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}
	enricher, err := NewEnricher(service)
	if err != nil {
		t.Fatalf("NewEnricher() err=%v", err)
	}

	result, err := enricher.MatchTrace(context.Background(), MatchTraceRequest{Profile: "driving"})
	if err != nil {
		t.Fatalf("MatchTrace() err=%v", err)
	}
	if result.Provider != "fake" || result.Profile != "driving" || result.Status != "matched" {
		t.Fatalf("MatchTrace()=%+v", result)
	}
}

type fakeEnrichmentProvider struct {
	name string
}

func (p *fakeEnrichmentProvider) Name() string { return p.name }
func (p *fakeEnrichmentProvider) Health(ctx context.Context) error { return nil }
func (p *fakeEnrichmentProvider) Capabilities(ctx context.Context) Capabilities {
	return Capabilities{MatchTrace: true}
}
func (p *fakeEnrichmentProvider) Route(ctx context.Context, req RouteRequest) (RouteResult, error) {
	return RouteResult{}, ErrNotImplemented
}
func (p *fakeEnrichmentProvider) MatchTrace(ctx context.Context, req MatchTraceRequest) (MatchTraceResult, error) {
	return MatchTraceResult{Provider: p.name, Profile: req.Profile, Status: "matched"}, nil
}
func (p *fakeEnrichmentProvider) Snap(ctx context.Context, req SnapRequest) (SnapResult, error) {
	return SnapResult{}, ErrNotImplemented
}
func (p *fakeEnrichmentProvider) Matrix(ctx context.Context, req MatrixRequest) (MatrixResult, error) {
	return MatrixResult{}, ErrNotImplemented
}
