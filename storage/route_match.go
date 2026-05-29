package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MickMake/GoTravel/routing"
)

// RouteMatchRun is a persisted route match run with linked source point IDs.
type RouteMatchRun struct {
	ID                int64
	Trace             routing.EnrichedTrace
	PointIDs          []int64
	CreatedAt         time.Time
	SourceFilterStart *time.Time
	SourceFilterEnd   *time.Time
}

// SaveRouteMatchRun persists an enriched trace and links it to the source points in order.
func (s *DB) SaveRouteMatchRun(ctx context.Context, trace routing.EnrichedTrace, pointIDs []int64, sourceStart, sourceEnd *time.Time) (runID int64, err error) {
	if len(pointIDs) != trace.SourcePointCount {
		return 0, fmt.Errorf("storage route match: %d point IDs for %d source points", len(pointIDs), trace.SourcePointCount)
	}
	if len(pointIDs) < 2 {
		return 0, fmt.Errorf("storage route match: at least two point IDs are required")
	}

	warningsJSON, err := json.Marshal(trace.Warnings)
	if err != nil {
		return 0, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	createdAt := time.Now().UTC()
	res, err := tx.ExecContext(ctx, `INSERT INTO route_match_runs (
provider, profile, status, source_point_count, geometry, geometry_format, distance_meters, duration_seconds,
confidence, warnings_json, raw_response, matched_at, created_at, source_filter_start, source_filter_end
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		trace.Provider,
		trace.Profile,
		trace.Status,
		trace.SourcePointCount,
		trace.Geometry,
		trace.GeometryFormat,
		trace.DistanceMeters,
		trace.DurationSeconds,
		nullFloat64(trace.Confidence),
		string(warningsJSON),
		trace.RawResponse,
		trace.MatchedAt.Format(timeLayout),
		createdAt.Format(timeLayout),
		nullTime(sourceStart),
		nullTime(sourceEnd),
	)
	if err != nil {
		return 0, err
	}
	runID, err = res.LastInsertId()
	if err != nil {
		return 0, err
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO route_match_points (route_match_run_id, point_id, sequence) VALUES (?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	for i, pointID := range pointIDs {
		if pointID <= 0 {
			return 0, fmt.Errorf("storage route match: point ID at sequence %d must be positive", i)
		}
		if _, err = stmt.ExecContext(ctx, runID, pointID, i); err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return runID, nil
}

// GetRouteMatchRun returns a persisted route match run by ID.
func (s *DB) GetRouteMatchRun(ctx context.Context, id int64) (RouteMatchRun, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, provider, profile, status, source_point_count, geometry, geometry_format,
distance_meters, duration_seconds, confidence, warnings_json, raw_response, matched_at, created_at, source_filter_start, source_filter_end
FROM route_match_runs WHERE id = ?`, id)

	var run RouteMatchRun
	var confidence sql.NullFloat64
	var warningsJSON string
	var matchedAt, createdAt string
	var sourceStart, sourceEnd sql.NullString
	err := row.Scan(
		&run.ID,
		&run.Trace.Provider,
		&run.Trace.Profile,
		&run.Trace.Status,
		&run.Trace.SourcePointCount,
		&run.Trace.Geometry,
		&run.Trace.GeometryFormat,
		&run.Trace.DistanceMeters,
		&run.Trace.DurationSeconds,
		&confidence,
		&warningsJSON,
		&run.Trace.RawResponse,
		&matchedAt,
		&createdAt,
		&sourceStart,
		&sourceEnd,
	)
	if err != nil {
		return RouteMatchRun{}, err
	}
	if confidence.Valid {
		value := confidence.Float64
		run.Trace.Confidence = &value
	}
	if warningsJSON != "" {
		if err := json.Unmarshal([]byte(warningsJSON), &run.Trace.Warnings); err != nil {
			return RouteMatchRun{}, err
		}
	}
	if run.Trace.MatchedAt, err = time.ParseInLocation(timeLayout, matchedAt, time.Local); err != nil {
		return RouteMatchRun{}, err
	}
	if run.CreatedAt, err = time.ParseInLocation(timeLayout, createdAt, time.Local); err != nil {
		return RouteMatchRun{}, err
	}
	if run.SourceFilterStart, err = parseNullTime(sourceStart); err != nil {
		return RouteMatchRun{}, err
	}
	if run.SourceFilterEnd, err = parseNullTime(sourceEnd); err != nil {
		return RouteMatchRun{}, err
	}

	pointRows, err := s.db.QueryContext(ctx, `SELECT point_id FROM route_match_points WHERE route_match_run_id = ? ORDER BY sequence`, id)
	if err != nil {
		return RouteMatchRun{}, err
	}
	defer pointRows.Close()
	for pointRows.Next() {
		var pointID int64
		if err := pointRows.Scan(&pointID); err != nil {
			return RouteMatchRun{}, err
		}
		run.PointIDs = append(run.PointIDs, pointID)
	}
	if err := pointRows.Err(); err != nil {
		return RouteMatchRun{}, err
	}
	return run, nil
}

func nullFloat64(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *value, Valid: true}
}

func nullTime(value *time.Time) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: value.Format(timeLayout), Valid: true}
}

func parseNullTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid || value.String == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation(timeLayout, value.String, time.Local)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
