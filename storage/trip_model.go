package storage

import "time"

const DefaultTripGapMinutes = 30

// Trip is a persisted conservative segmentation of staged GPS points.
type Trip struct {
	ID               int64
	StartTime        time.Time
	EndTime          time.Time
	SourcePointCount int
	FirstPointID     int64
	LastPointID      int64
	DurationSeconds  int64
	GapSeconds       int64
	CreatedAt        time.Time
	PointIDs         []int64
}

// SegmentTripsOptions controls trip segmentation over staged points.
type SegmentTripsOptions struct {
	Gap   time.Duration
	Force bool
	Now   func() time.Time
}

// SegmentPointsByGap groups staged points into deterministic trips using only time gaps.
func SegmentPointsByGap(points []Point, gap time.Duration) []Trip {
	if gap <= 0 {
		gap = time.Duration(DefaultTripGapMinutes) * time.Minute
	}
	if len(points) < 2 {
		return nil
	}

	var trips []Trip
	start := 0
	for i := 1; i < len(points); i++ {
		if points[i].DT.Sub(points[i-1].DT) > gap {
			trips = appendTripSegment(trips, points[start:i], gap)
			start = i
		}
	}
	trips = appendTripSegment(trips, points[start:], gap)
	return trips
}

func appendTripSegment(trips []Trip, points []Point, gap time.Duration) []Trip {
	if len(points) < 2 {
		return trips
	}
	first := points[0]
	last := points[len(points)-1]
	pointIDs := make([]int64, 0, len(points))
	for _, point := range points {
		pointIDs = append(pointIDs, point.ID)
	}
	return append(trips, Trip{
		StartTime:        first.DT,
		EndTime:          last.DT,
		SourcePointCount: len(points),
		FirstPointID:     first.ID,
		LastPointID:      last.ID,
		DurationSeconds:  int64(last.DT.Sub(first.DT).Seconds()),
		GapSeconds:       int64(gap.Seconds()),
		PointIDs:         pointIDs,
	})
}
