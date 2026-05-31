package storage

import (
	"testing"
	"time"
)

func TestSegmentPointsByGapNoPoints(t *testing.T) {
	trips := SegmentPointsByGap(nil, 30*time.Minute)
	if len(trips) != 0 {
		t.Fatalf("len(trips)=%d want 0", len(trips))
	}
}

func TestSegmentPointsByGapOnePoint(t *testing.T) {
	points := []Point{{ID: 1, DT: tripTestTime(0)}}
	trips := SegmentPointsByGap(points, 30*time.Minute)
	if len(trips) != 0 {
		t.Fatalf("len(trips)=%d want 0", len(trips))
	}
}

func TestSegmentPointsByGapUnderThresholdStaysOneTrip(t *testing.T) {
	points := []Point{
		{ID: 1, DT: tripTestTime(0)},
		{ID: 2, DT: tripTestTime(10 * time.Minute)},
		{ID: 3, DT: tripTestTime(20 * time.Minute)},
	}
	trips := SegmentPointsByGap(points, 30*time.Minute)
	if len(trips) != 1 {
		t.Fatalf("len(trips)=%d want 1", len(trips))
	}
	trip := trips[0]
	if trip.SourcePointCount != 3 || trip.FirstPointID != 1 || trip.LastPointID != 3 {
		t.Fatalf("unexpected trip metadata: %+v", trip)
	}
	if !trip.StartTime.Equal(points[0].DT) || !trip.EndTime.Equal(points[2].DT) {
		t.Fatalf("unexpected trip times: %+v", trip)
	}
	if trip.DurationSeconds != int64((20*time.Minute).Seconds()) {
		t.Fatalf("duration=%d", trip.DurationSeconds)
	}
	if got := trip.PointIDs; len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("point IDs=%+v", got)
	}
}

func TestSegmentPointsByGapOverThresholdSplitsTrips(t *testing.T) {
	points := []Point{
		{ID: 1, DT: tripTestTime(0)},
		{ID: 2, DT: tripTestTime(10 * time.Minute)},
		{ID: 3, DT: tripTestTime(45 * time.Minute)},
		{ID: 4, DT: tripTestTime(50 * time.Minute)},
	}
	trips := SegmentPointsByGap(points, 30*time.Minute)
	if len(trips) != 2 {
		t.Fatalf("len(trips)=%d want 2", len(trips))
	}
	if trips[0].FirstPointID != 1 || trips[0].LastPointID != 2 {
		t.Fatalf("first trip=%+v", trips[0])
	}
	if trips[1].FirstPointID != 3 || trips[1].LastPointID != 4 {
		t.Fatalf("second trip=%+v", trips[1])
	}
}

func TestSegmentPointsByGapExactThresholdStaysOneTrip(t *testing.T) {
	points := []Point{
		{ID: 1, DT: tripTestTime(0)},
		{ID: 2, DT: tripTestTime(30 * time.Minute)},
	}
	trips := SegmentPointsByGap(points, 30*time.Minute)
	if len(trips) != 1 {
		t.Fatalf("len(trips)=%d want 1", len(trips))
	}
}

func tripTestTime(offset time.Duration) time.Time {
	return time.Unix(1700000000, 0).UTC().Add(offset)
}
