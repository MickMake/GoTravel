package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSegmentTripsPersistsTripsAndMembership(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()
	ids := insertTripTestPoints(t, store, []time.Duration{0, 10 * time.Minute, 45 * time.Minute, 50 * time.Minute})
	createdAt := time.Unix(1700009000, 0).UTC()

	trips, err := store.SegmentTrips(context.Background(), SegmentTripsOptions{Gap: 30 * time.Minute, Now: func() time.Time { return createdAt }})
	if err != nil {
		t.Fatalf("SegmentTrips() err=%v", err)
	}
	if len(trips) != 2 {
		t.Fatalf("len(trips)=%d want 2", len(trips))
	}
	if trips[0].ID <= 0 || trips[1].ID <= 0 {
		t.Fatalf("trip IDs not set: %+v", trips)
	}
	if trips[0].SourcePointCount != 2 || trips[0].FirstPointID != ids[0] || trips[0].LastPointID != ids[1] {
		t.Fatalf("first trip metadata=%+v ids=%+v", trips[0], ids)
	}
	if trips[1].SourcePointCount != 2 || trips[1].FirstPointID != ids[2] || trips[1].LastPointID != ids[3] {
		t.Fatalf("second trip metadata=%+v ids=%+v", trips[1], ids)
	}

	stored, err := store.GetTrip(context.Background(), trips[0].ID)
	if err != nil {
		t.Fatalf("GetTrip() err=%v", err)
	}
	if len(stored.PointIDs) != 2 || stored.PointIDs[0] != ids[0] || stored.PointIDs[1] != ids[1] {
		t.Fatalf("stored point IDs=%+v want first two IDs %+v", stored.PointIDs, ids[:2])
	}
	if !stored.CreatedAt.Equal(createdAt) {
		t.Fatalf("createdAt=%v want %v", stored.CreatedAt, createdAt)
	}
}

func TestSegmentTripsNoPointsAndOnePointPersistNoTrips(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()

	trips, err := store.SegmentTrips(context.Background(), SegmentTripsOptions{Gap: 30 * time.Minute})
	if err != nil {
		t.Fatalf("SegmentTrips() no points err=%v", err)
	}
	if len(trips) != 0 {
		t.Fatalf("len(trips)=%d want 0", len(trips))
	}
	insertTripTestPoints(t, store, []time.Duration{0})
	trips, err = store.SegmentTrips(context.Background(), SegmentTripsOptions{Gap: 30 * time.Minute, Force: true})
	if err != nil {
		t.Fatalf("SegmentTrips() one point err=%v", err)
	}
	if len(trips) != 0 {
		t.Fatalf("len(trips)=%d want 0", len(trips))
	}
}

func TestSegmentTripsRepeatedRunRequiresForce(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()
	insertTripTestPoints(t, store, []time.Duration{0, 5 * time.Minute})

	if _, err := store.SegmentTrips(context.Background(), SegmentTripsOptions{Gap: 30 * time.Minute}); err != nil {
		t.Fatalf("SegmentTrips() first run err=%v", err)
	}
	_, err = store.SegmentTrips(context.Background(), SegmentTripsOptions{Gap: 30 * time.Minute})
	if err == nil || !strings.Contains(err.Error(), "use --force") {
		t.Fatalf("expected repeat-run force error, got %v", err)
	}
	trips, err := store.SegmentTrips(context.Background(), SegmentTripsOptions{Gap: 30 * time.Minute, Force: true})
	if err != nil {
		t.Fatalf("SegmentTrips() force err=%v", err)
	}
	if len(trips) != 1 {
		t.Fatalf("len(trips)=%d want 1", len(trips))
	}
	listed, err := store.ListTrips(context.Background())
	if err != nil {
		t.Fatalf("ListTrips() err=%v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed trips=%d want 1", len(listed))
	}
}

func TestGetTripMissing(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()
	_, err = store.GetTrip(context.Background(), 999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err=%v want sql.ErrNoRows", err)
	}
}

func insertTripTestPoints(t *testing.T, store *DB, offsets []time.Duration) []int64 {
	t.Helper()
	ids := make([]int64, 0, len(offsets))
	base := time.Unix(1700000000, 0).UTC()
	for i, offset := range offsets {
		point := Point{DT: base.Add(offset), Lat: -33.8 + float64(i)*0.01, Lng: 151.2 + float64(i)*0.01, Format: "test", SourceFile: "trip.csv", SourceLine: i + 1, ImportedAt: base.Add(offset).Add(time.Second)}
		point.PointHash = PointHash(point)
		res, err := store.db.Exec(`INSERT INTO points (dt, lat, lng, altitude, angle, speed, params, format, source_file, source_line, imported_at, point_hash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, point.DT.Format(timeLayout), point.Lat, point.Lng, point.Altitude, point.Angle, point.Speed, point.Params, point.Format, point.SourceFile, point.SourceLine, point.ImportedAt.Format(timeLayout), point.PointHash)
		if err != nil {
			t.Fatalf("insert test point: %v", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("LastInsertId() err=%v", err)
		}
		ids = append(ids, id)
	}
	return ids
}
