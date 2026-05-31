package storage

import (
	"context"
	"encoding/json"
	"fmt"
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
	if provider.pointCounts[0] != 2 || provider.profile != "driving" {
		t.Fatalf("provider saw profile=%q pointCounts=%v", provider.profile, provider.pointCounts)
	}
}

func TestRemoveConsecutiveDuplicateCoordinates(t *testing.T) {
	points := []Point{
		{ID: 1, DT: time.Unix(1, 0), Lat: -33.8, Lng: 151.2},
		{ID: 2, DT: time.Unix(2, 0), Lat: -33.8, Lng: 151.2},
		{ID: 3, DT: time.Unix(3, 0), Lat: -33.9, Lng: 151.3},
		{ID: 4, DT: time.Unix(4, 0), Lat: -33.8, Lng: 151.2},
	}

	cleaned := removeConsecutiveDuplicateCoordinates(points)
	if len(cleaned) != 3 {
		t.Fatalf("cleaned len=%d want 3", len(cleaned))
	}
	wantIDs := []int64{1, 3, 4}
	for i, wantID := range wantIDs {
		if cleaned[i].ID != wantID {
			t.Fatalf("cleaned[%d].ID=%d want %d", i, cleaned[i].ID, wantID)
		}
	}
	if !cleaned[1].DT.Equal(points[2].DT) {
		t.Fatalf("timestamp not preserved: %v", cleaned[1].DT)
	}
}

func TestRouteMatchTraceChunksUnderLimit(t *testing.T) {
	points := routeMatchTestPoints(3)
	chunks := routeMatchTraceChunks(points, 100, 1)
	if len(chunks) != 1 || len(chunks[0]) != 3 {
		t.Fatalf("chunks=%v", chunkLengths(chunks))
	}
}

func TestRouteMatchTraceChunksOverLimit(t *testing.T) {
	points := routeMatchTestPoints(201)
	chunks := routeMatchTraceChunks(points, 100, 1)
	if got := chunkLengths(chunks); fmt.Sprint(got) != "[100 100 3]" {
		t.Fatalf("chunk lengths=%v", got)
	}
	if chunks[0][99].ID != 100 || chunks[1][0].ID != 100 || chunks[1][99].ID != 199 || chunks[2][0].ID != 199 {
		t.Fatalf("unexpected overlap boundaries")
	}
}

func TestRouteMatchTraceChunksOverlapBehaviour(t *testing.T) {
	points := routeMatchTestPoints(5)
	chunks := routeMatchTraceChunks(points, 3, 1)
	if got := chunkLengths(chunks); fmt.Sprint(got) != "[3 3]" {
		t.Fatalf("chunk lengths=%v", got)
	}
	if chunks[0][2].ID != chunks[1][0].ID {
		t.Fatalf("expected one-point overlap, got %d and %d", chunks[0][2].ID, chunks[1][0].ID)
	}
}

func TestRouteMatchTraceChunksRejectUnsafeSmallInputs(t *testing.T) {
	if chunks := routeMatchTraceChunks(routeMatchTestPoints(1), 100, 1); len(chunks) != 0 {
		t.Fatalf("single-point chunks=%v", chunks)
	}
	if chunks := routeMatchTraceChunks(routeMatchTestPoints(3), 1, 1); len(chunks) != 0 {
		t.Fatalf("maxPoints<2 chunks=%v", chunks)
	}
}

func TestRouteMatchRunnerChunksAndCombinesGeometry(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()

	points := routeMatchTestPoints(102)
	points[1].Lat = points[0].Lat
	points[1].Lng = points[0].Lng
	saveRouteMatchTestPoints(t, store, points)

	provider := &runnerProvider{provider: "fake", geoJSON: true}
	run := runRouteMatchWithProvider(t, store, provider)

	if provider.calls != 2 {
		t.Fatalf("provider calls=%d want 2", provider.calls)
	}
	if got := fmt.Sprint(provider.pointCounts); got != "[100 2]" {
		t.Fatalf("provider point counts=%s", got)
	}
	if run.Trace.SourcePointCount != 101 {
		t.Fatalf("source count=%d want 101", run.Trace.SourcePointCount)
	}
	if len(run.PointIDs) != 101 {
		t.Fatalf("linked point IDs=%d want 101", len(run.PointIDs))
	}
	if run.Trace.GeometryFormat != "geojson" {
		t.Fatalf("geometry format=%q", run.Trace.GeometryFormat)
	}
	coordinates, err := routing.RouteGeometryCoordinates(run.Trace.GeometryFormat, run.Trace.Geometry)
	if err != nil {
		t.Fatalf("RouteGeometryCoordinates() err=%v", err)
	}
	if len(coordinates) != 101 {
		t.Fatalf("combined coordinates=%d want 101", len(coordinates))
	}
	if run.Trace.DistanceMeters != 102 || run.Trace.DurationSeconds != 3 {
		t.Fatalf("combined metrics distance=%v duration=%v", run.Trace.DistanceMeters, run.Trace.DurationSeconds)
	}
	if len(run.Trace.RawResponse) == 0 || !json.Valid(run.Trace.RawResponse) {
		t.Fatalf("combined raw response is not JSON: %q", string(run.Trace.RawResponse))
	}
}

func TestRouteMatchRunnerChunkFailurePreventsStorage(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()

	saveRouteMatchTestPoints(t, store, routeMatchTestPoints(101))
	provider := &runnerProvider{provider: "fake", geoJSON: true, failOnCall: 2}
	service, err := routing.NewService(provider)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}
	enricher, err := routing.NewEnricher(service)
	if err != nil {
		t.Fatalf("NewEnricher() err=%v", err)
	}

	_, err = (RouteMatchRunner{Store: store, Enricher: enricher}).RunMatchTrace(context.Background(), RouteMatchRunOptions{Profile: "driving"})
	if err == nil || !strings.Contains(err.Error(), "chunk 2/2 provider match failed") {
		t.Fatalf("err=%v", err)
	}
	if _, getErr := store.GetRouteMatchRun(context.Background(), 1); getErr == nil {
		t.Fatalf("expected no successful stored run after chunk failure")
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
	provider    string
	profile     string
	pointCounts []int
	calls       int
	failOnCall  int
	geoJSON     bool
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
	p.calls++
	p.profile = req.Profile
	p.pointCounts = append(p.pointCounts, len(req.Points))
	if p.failOnCall == p.calls {
		return routing.MatchTraceResult{}, fmt.Errorf("boom")
	}

	geometry := "matched"
	geometryFormat := "polyline6"
	if p.geoJSON {
		geometry = traceRequestGeoJSON(req)
		geometryFormat = "geojson"
	}
	return routing.MatchTraceResult{
		Provider:        p.provider,
		Profile:         req.Profile,
		Status:          "Ok",
		Geometry:        geometry,
		GeometryFormat:  geometryFormat,
		DistanceMeters:  float64(len(req.Points)),
		DurationSeconds: float64(p.calls),
		RawResponse:     []byte(fmt.Sprintf(`{"chunk":%d}`, p.calls)),
	}, nil
}
func (p *runnerProvider) Snap(ctx context.Context, req routing.SnapRequest) (routing.SnapResult, error) {
	return routing.SnapResult{}, routing.ErrNotImplemented
}
func (p *runnerProvider) Matrix(ctx context.Context, req routing.MatrixRequest) (routing.MatrixResult, error) {
	return routing.MatrixResult{}, routing.ErrNotImplemented
}

func routeMatchTestPoints(count int) []Point {
	points := make([]Point, 0, count)
	for i := 0; i < count; i++ {
		points = append(points, Point{
			ID:         int64(i + 1),
			DT:         time.Unix(1700000000+int64(i*60), 0),
			Lat:        -33.8 + float64(i)*0.001,
			Lng:        151.2 + float64(i)*0.001,
			Format:     "test",
			SourceFile: "test.csv",
			SourceLine: i + 1,
		})
	}
	return points
}

func saveRouteMatchTestPoints(t *testing.T, store *DB, points []Point) {
	t.Helper()
	if _, _, err := store.SaveImportResult(ImportResult{Format: "test", SourceFile: "test.csv", RowsSeen: len(points), Points: points}, false); err != nil {
		t.Fatalf("SaveImportResult() err=%v", err)
	}
}

func runRouteMatchWithProvider(t *testing.T, store *DB, provider *runnerProvider) RouteMatchRun {
	t.Helper()
	service, err := routing.NewService(provider)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}
	enricher, err := routing.NewEnricher(service)
	if err != nil {
		t.Fatalf("NewEnricher() err=%v", err)
	}
	run, err := (RouteMatchRunner{Store: store, Enricher: enricher}).RunMatchTrace(context.Background(), RouteMatchRunOptions{Profile: "driving"})
	if err != nil {
		t.Fatalf("RunMatchTrace() err=%v", err)
	}
	return run
}

func chunkLengths(chunks [][]Point) []int {
	lengths := make([]int, 0, len(chunks))
	for _, chunk := range chunks {
		lengths = append(lengths, len(chunk))
	}
	return lengths
}

func traceRequestGeoJSON(req routing.MatchTraceRequest) string {
	coordinates := make([][]float64, 0, len(req.Points))
	for _, point := range req.Points {
		coordinates = append(coordinates, []float64{point.Lng, point.Lat})
	}
	encoded, _ := json.Marshal(map[string]any{
		"type":        "LineString",
		"coordinates": coordinates,
	})
	return string(encoded)
}
