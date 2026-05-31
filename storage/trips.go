package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

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

// SegmentTrips segments all staged points into conservative time-gap trips and persists them.
func (s *DB) SegmentTrips(ctx context.Context, options SegmentTripsOptions) ([]Trip, error) {
	gap := options.Gap
	if gap <= 0 {
		gap = time.Duration(DefaultTripGapMinutes) * time.Minute
	}

	points, err := s.QueryPoints(nil, nil)
	if err != nil {
		return nil, err
	}
	segments := SegmentPointsByGap(points, gap)

	createdAt := time.Now().UTC()
	if options.Now != nil {
		createdAt = options.Now().UTC()
	}
	return s.replaceTrips(ctx, segments, gap, createdAt, options.Force)
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

func (s *DB) replaceTrips(ctx context.Context, trips []Trip, gap time.Duration, createdAt time.Time, force bool) (saved []Trip, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var existing int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM trips`).Scan(&existing); err != nil {
		return nil, err
	}
	if existing > 0 && !force {
		return nil, fmt.Errorf("trip segmentation already exists; use --force to rebuild trips")
	}
	if force {
		if _, err = tx.ExecContext(ctx, `DELETE FROM trip_points`); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM trips`); err != nil {
			return nil, err
		}
	}

	for _, trip := range trips {
		trip.GapSeconds = int64(gap.Seconds())
		trip.CreatedAt = createdAt
		res, err := tx.ExecContext(ctx, `INSERT INTO trips (
start_time, end_time, source_point_count, first_point_id, last_point_id, duration_seconds, gap_seconds, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			trip.StartTime.Format(timeLayout),
			trip.EndTime.Format(timeLayout),
			trip.SourcePointCount,
			trip.FirstPointID,
			trip.LastPointID,
			trip.DurationSeconds,
			trip.GapSeconds,
			trip.CreatedAt.Format(timeLayout),
		)
		if err != nil {
			return nil, err
		}
		trip.ID, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}

		stmt, err := tx.PrepareContext(ctx, `INSERT INTO trip_points (trip_id, point_id, sequence) VALUES (?, ?, ?)`) 
		if err != nil {
			return nil, err
		}
		for i, pointID := range trip.PointIDs {
			if pointID <= 0 {
				_ = stmt.Close()
				return nil, fmt.Errorf("trip point ID at sequence %d must be positive", i)
			}
			if _, err = stmt.ExecContext(ctx, trip.ID, pointID, i); err != nil {
				_ = stmt.Close()
				return nil, err
			}
		}
		if err = stmt.Close(); err != nil {
			return nil, err
		}
		saved = append(saved, trip)
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return saved, nil
}

// ListTrips returns persisted trips ordered by start time.
func (s *DB) ListTrips(ctx context.Context) ([]Trip, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, start_time, end_time, source_point_count, first_point_id, last_point_id, duration_seconds, gap_seconds, created_at FROM trips ORDER BY start_time, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []Trip
	for rows.Next() {
		trip, err := scanTripRows(rows)
		if err != nil {
			return nil, err
		}
		trips = append(trips, trip)
	}
	return trips, rows.Err()
}

// GetTrip returns one persisted trip by ID, including linked source point IDs.
func (s *DB) GetTrip(ctx context.Context, id int64) (Trip, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, start_time, end_time, source_point_count, first_point_id, last_point_id, duration_seconds, gap_seconds, created_at FROM trips WHERE id = ?`, id)
	trip, err := scanTripRow(row)
	if err != nil {
		return Trip{}, err
	}

	pointRows, err := s.db.QueryContext(ctx, `SELECT point_id FROM trip_points WHERE trip_id = ? ORDER BY sequence`, id)
	if err != nil {
		return Trip{}, err
	}
	defer pointRows.Close()
	for pointRows.Next() {
		var pointID int64
		if err := pointRows.Scan(&pointID); err != nil {
			return Trip{}, err
		}
		trip.PointIDs = append(trip.PointIDs, pointID)
	}
	if err := pointRows.Err(); err != nil {
		return Trip{}, err
	}
	return trip, nil
}

type tripScanner interface {
	Scan(dest ...any) error
}

func scanTripRows(rows *sql.Rows) (Trip, error) { return scanTripRow(rows) }

func scanTripRow(row tripScanner) (Trip, error) {
	var trip Trip
	var startTime, endTime, createdAt string
	if err := row.Scan(&trip.ID, &startTime, &endTime, &trip.SourcePointCount, &trip.FirstPointID, &trip.LastPointID, &trip.DurationSeconds, &trip.GapSeconds, &createdAt); err != nil {
		return Trip{}, err
	}
	var err error
	if trip.StartTime, err = time.ParseInLocation(timeLayout, startTime, time.UTC); err != nil {
		return Trip{}, err
	}
	if trip.EndTime, err = time.ParseInLocation(timeLayout, endTime, time.UTC); err != nil {
		return Trip{}, err
	}
	if trip.CreatedAt, err = time.ParseInLocation(timeLayout, createdAt, time.UTC); err != nil {
		return Trip{}, err
	}
	return trip, nil
}
