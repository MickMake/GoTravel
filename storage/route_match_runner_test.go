package storage

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MickMake/GoTravel/routing"
)

func TestRouteMatchRunnerRunMatchTrace(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()

	points := []Point{
		{DT: time.Unix(1700000000, 0), Lat: -33.8, Lng: 151.2, Format: "test", SourceFile: "test.csv", SourceLine: 1},
		{DT: time.Unix(1700000060, 0), Lat: -33.9, Lng: 151.3, Format: "test", SourceFile: "test.csv", SourceLine: 2},
	}
	if _, _, err := store.SaveImportResult(ImportResult{Format: "test", SourceFile: "test.csv", RowsSeen: len(points), Points: points}, false); err != nil {
		t.Fatalf("SaveImportResult() err=%v", err)
	}

	provider := &runnerProvider{provider: "fake"}
	service, err := routing.NewService(provider)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}
	enricher, err := routing.NewEnricher(service)
	if err != nil {
		t.Fatalf("NewEnricher() err=%v", err)
	}

	matchedAt := time.Unix(1700001000, 0)
	runner := RouteMatchRunner{Store: store, Enricher: enricher, Now: func() time.Time { return matchedAt }}
	run, err := runner.RunMatchTrace(context.Background(), RouteMatchRunOptions{Profile: "driving"})
	if err != nil {
		t.Fatalf("RunMatchTrace() err=%v", err)
	}
	if run.ID <= 0 {
		t.Fatalf("run ID=%d", run.ID)
	}
	if run.Trace.Provider != "fake" || run.Trace.Profile != "driving" || run.Trace.Status != "Ok" {
		t.Fatalf("trace=%+v", run.Trace)
	}
	if run.Trace.SourcePointCount != 2 || run.Trace.Geometry != "matched" || run.Trace.GeometryFormat != "polyline6" {
		t.Fatalf("trace metadata=%+v", run.Trace)
	}
	if run.Trace.DistanceMeters != 123 || run.Trace.DurationSeconds != 45 {
		t.Fatalf("trace metrics=%+v", run.Trace)
	}
	if !run.Trace.MatchedAt.Equal(matchedAt) {
		t.Fatalf("matchedAt=%v want %v", run.Trace.MatchedAt, matchedAt)
	}
	if len(run.PointIDs) != 2 || run.PointIDs[0] <= 0 || run.PointIDs[1] <= 0 {
		t.Fatalf("point IDs=%+v", run.PointIDs)
	}
	if provider.pointCount != 2 || provider.profile != "driving" {
		t.Fatalf("provider saw profile=%q pointCount=%d", provider.profile, provider.pointCount)
	}
}

func TestRouteMatchRunnerRequiresStoreAndEnricher(t *testing.T) {
	_, err := (RouteMatchRunner{}).RunMatchTrace(context.Background(), RouteMatchRunOptions{})
	if err == nil || !strings.Contains(err.Error(), "store is required") {
		t.Fatalf("missing store err=%v", err)
	}

	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()

	_, err = (RouteMatchRunner{Store: store}).RunMatchTrace(context.Background(), RouteMatchRunOptions{})
	if err == nil || !strings.Contains(err.Error(), "enricher is required") {
		t.Fatalf("missing enricher err=%v", err)
	}
}

type runnerProvider struct {
	provider   string
	profile    string
	pointCount int
}

func (p *runnerProvider) Name() string                     { return p.provider }
func (p *runnerProvider) Health(ctx context.Context) error { return nil }
func (p *runnerProvider) Capabilities(ctx context.Context) routing.Capabilities {
	return routing.Capabilities{MatchTrace: true}
}
func (p *runnerProvider) Route(ctx context.Context, req routing.RouteRequest) (routing.RouteResult, error) {
	return routing.RouteResult{}, routing.ErrNotImplemented
}
func (p *runnerProvider) MatchTrace(ctx context.Context, req routing.MatchTraceRequest) (routing.MatchTraceResult, error) {
	p.profile = req.Profile
	p.pointCount = len(req.Points)
	return routing.MatchTraceResult{
		Provider:        p.provider,
		Profile:         req.Profile,
		Status:          "Ok",
		Geometry:        "matched",
		GeometryFormat:  "polyline6",
		DistanceMeters:  123,
		DurationSeconds: 45,
		RawResponse:     []byte(`{"code":"Ok"}`),
	}, nil
}
func (p *runnerProvider) Snap(ctx context.Context, req routing.SnapRequest) (routing.SnapResult, error) {
	return routing.SnapResult{}, routing.ErrNotImplemented
}
func (p *runnerProvider) Matrix(ctx context.Context, req routing.MatrixRequest) (routing.MatrixResult, error) {
	return routing.MatrixResult{}, routing.ErrNotImplemented
}
