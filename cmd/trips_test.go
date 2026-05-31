package cmd

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MickMake/GoTravel/storage"
)

func TestRunTripsSegmentAndInspect(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "gotravel.sqlite")
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	base := time.Unix(1700000000, 0).UTC()
	result := storage.ImportResult{
		Format:     "test",
		SourceFile: "trip.csv",
		RowsSeen:   4,
		Points: []storage.Point{
			{DT: base, Lat: -33.80, Lng: 151.20, Format: "test", SourceFile: "trip.csv", SourceLine: 1},
			{DT: base.Add(10 * time.Minute), Lat: -33.81, Lng: 151.21, Format: "test", SourceFile: "trip.csv", SourceLine: 2},
			{DT: base.Add(45 * time.Minute), Lat: -33.82, Lng: 151.22, Format: "test", SourceFile: "trip.csv", SourceLine: 3},
			{DT: base.Add(50 * time.Minute), Lat: -33.83, Lng: 151.23, Format: "test", SourceFile: "trip.csv", SourceLine: 4},
		},
	}
	if _, _, err := store.SaveImportResult(result, false); err != nil {
		t.Fatalf("SaveImportResult() err=%v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() err=%v", err)
	}

	if err := runTrips([]string{"segment", "--db", dbPath, "--gap-minutes", "30"}); err != nil {
		t.Fatalf("runTrips segment err=%v", err)
	}
	store, err = storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open() after segment err=%v", err)
	}
	trips, err := store.ListTrips(context.Background())
	if err != nil {
		t.Fatalf("ListTrips() err=%v", err)
	}
	if len(trips) != 2 {
		t.Fatalf("len(trips)=%d want 2", len(trips))
	}
	firstID := trips[0].ID
	if err := store.Close(); err != nil {
		t.Fatalf("Close() after list err=%v", err)
	}
	if err := runTrips([]string{"inspect", "--db", dbPath, strconv.FormatInt(firstID, 10)}); err != nil {
		t.Fatalf("runTrips inspect err=%v", err)
	}
}

func TestRunTripsSegmentRejectsInvalidGap(t *testing.T) {
	err := runTrips([]string{"segment", "--gap-minutes", "0"})
	if err == nil || !strings.Contains(err.Error(), "gap-minutes must be positive") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunTripsUnknownCommand(t *testing.T) {
	err := runTrips([]string{"summon"})
	if err == nil || !strings.Contains(err.Error(), "unknown trips command") {
		t.Fatalf("err=%v", err)
	}
}
