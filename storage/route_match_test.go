package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MickMake/GoTravel/routing"
)

func TestSaveAndGetRouteMatchRun(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()

	confidence := 0.88
	matchedAt := time.Unix(1700001000, 0).UTC()
	start := time.Unix(1700000000, 0).UTC()
	end := time.Unix(1700000600, 0).UTC()
	trace := routing.EnrichedTrace{
		Provider:         "osrm",
		Profile:          "driving",
		Status:           "Ok",
		SourcePointCount: 2,
		Geometry:         "encoded",
		GeometryFormat:   "polyline6",
		DistanceMeters:   123.4,
		DurationSeconds:  56.7,
		Confidence:       &confidence,
		Warnings:         []string{"minor"},
		RawResponse:      []byte(`{"code":"Ok"}`),
		MatchedAt:        matchedAt,
	}

	runID, err := store.SaveRouteMatchRun(context.Background(), trace, []int64{10, 20}, &start, &end)
	if err != nil {
		t.Fatalf("SaveRouteMatchRun() err=%v", err)
	}
	if runID <= 0 {
		t.Fatalf("runID=%d", runID)
	}

	run, err := store.GetRouteMatchRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("GetRouteMatchRun() err=%v", err)
	}
	if run.ID != runID {
		t.Fatalf("ID=%d want %d", run.ID, runID)
	}
	if run.Trace.Provider != "osrm" || run.Trace.Profile != "driving" || run.Trace.Status != "Ok" {
		t.Fatalf("unexpected identity/status: %+v", run.Trace)
	}
	if run.Trace.SourcePointCount != 2 || run.Trace.Geometry != "encoded" || run.Trace.GeometryFormat != "polyline6" {
		t.Fatalf("unexpected metadata: %+v", run.Trace)
	}
	if run.Trace.DistanceMeters != 123.4 || run.Trace.DurationSeconds != 56.7 {
		t.Fatalf("unexpected metrics: %+v", run.Trace)
	}
	if run.Trace.Confidence == nil || *run.Trace.Confidence != confidence {
		t.Fatalf("confidence=%v", run.Trace.Confidence)
	}
	if len(run.Trace.Warnings) != 1 || run.Trace.Warnings[0] != "minor" {
		t.Fatalf("warnings=%+v", run.Trace.Warnings)
	}
	if string(run.Trace.RawResponse) != `{"code":"Ok"}` {
		t.Fatalf("raw=%s", string(run.Trace.RawResponse))
	}
	if !run.Trace.MatchedAt.Equal(matchedAt) {
		t.Fatalf("matchedAt=%v want %v", run.Trace.MatchedAt, matchedAt)
	}
	if len(run.PointIDs) != 2 || run.PointIDs[0] != 10 || run.PointIDs[1] != 20 {
		t.Fatalf("point IDs=%+v", run.PointIDs)
	}
	if run.SourceFilterStart == nil || !run.SourceFilterStart.Equal(start) {
		t.Fatalf("source start=%v", run.SourceFilterStart)
	}
	if run.SourceFilterEnd == nil || !run.SourceFilterEnd.Equal(end) {
		t.Fatalf("source end=%v", run.SourceFilterEnd)
	}
}

func TestSaveRouteMatchRunRejectsMismatchedPointIDs(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()

	trace := routing.EnrichedTrace{SourcePointCount: 2, MatchedAt: time.Unix(1700001000, 0)}
	_, err = store.SaveRouteMatchRun(context.Background(), trace, []int64{10}, nil, nil)
	if err == nil || !errors.Is(err, sql.ErrNoRows) && err.Error() == "" {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestGetRouteMatchRunMissing(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()

	_, err = store.GetRouteMatchRun(context.Background(), 999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err=%v want sql.ErrNoRows", err)
	}
}
