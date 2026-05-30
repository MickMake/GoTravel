package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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

	pointIDs := insertRouteMatchTestPoints(t, store)
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

	runID, err := store.SaveRouteMatchRun(context.Background(), trace, pointIDs, &start, &end)
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
	if len(run.PointIDs) != 2 || run.PointIDs[0] != pointIDs[0] || run.PointIDs[1] != pointIDs[1] {
		t.Fatalf("point IDs=%+v want %+v", run.PointIDs, pointIDs)
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
	if err == nil || !strings.Contains(err.Error(), "1 point IDs for 2 source points") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestSaveRouteMatchRunRejectsUnknownPointIDs(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()

	trace := routing.EnrichedTrace{SourcePointCount: 2, MatchedAt: time.Unix(1700001000, 0)}
	_, err = store.SaveRouteMatchRun(context.Background(), trace, []int64{10, 20}, nil, nil)
	if err == nil {
		t.Fatal("SaveRouteMatchRun accepted unknown point IDs")
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

func insertRouteMatchTestPoints(t *testing.T, store *DB) []int64 {
	t.Helper()
	points := []Point{
		{DT: time.Unix(1700000000, 0).UTC(), Lat: -33.8, Lng: 151.2, Format: "test", SourceFile: "test.csv", SourceLine: 1, ImportedAt: time.Unix(1700000001, 0).UTC()},
		{DT: time.Unix(1700000060, 0).UTC(), Lat: -33.9, Lng: 151.3, Format: "test", SourceFile: "test.csv", SourceLine: 2, ImportedAt: time.Unix(1700000061, 0).UTC()},
	}
	ids := make([]int64, 0, len(points))
	for _, point := range points {
		point.PointHash = PointHash(point)
		res, err := store.db.Exec(`INSERT INTO points (dt, lat, lng, altitude, angle, speed, params, format, source_file, source_line, imported_at, point_hash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			point.DT.Format(timeLayout), point.Lat, point.Lng, point.Altitude, point.Angle, point.Speed, point.Params,
			point.Format, point.SourceFile, point.SourceLine, point.ImportedAt.Format(timeLayout), point.PointHash,
		)
		if err != nil {
			t.Fatalf("insert test point: %v", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("test point id: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}
