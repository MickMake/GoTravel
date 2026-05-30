package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/MickMake/GoTravel/routing"
)

// RouteMatchRunner coordinates staged point loading, trace matching, and persistence.
type RouteMatchRunner struct {
	Store    *DB
	Enricher *routing.Enricher
	Now      func() time.Time
}

// RouteMatchRunOptions controls an internal route-match run.
type RouteMatchRunOptions struct {
	Profile string
	Radius  *float64
	Start   *time.Time
	End     *time.Time
}

// RunMatchTrace loads staged points, matches them through the enricher, and stores the enriched trace.
func (r RouteMatchRunner) RunMatchTrace(ctx context.Context, options RouteMatchRunOptions) (RouteMatchRun, error) {
	if r.Store == nil {
		return RouteMatchRun{}, fmt.Errorf("storage route match runner: store is required")
	}
	if r.Enricher == nil {
		return RouteMatchRun{}, fmt.Errorf("storage route match runner: enricher is required")
	}

	points, err := r.Store.QueryPoints(options.Start, options.End)
	if err != nil {
		return RouteMatchRun{}, err
	}
	request, err := MatchTraceRequestFromPoints(points, MatchTraceOptions{Profile: options.Profile, Radius: options.Radius})
	if err != nil {
		return RouteMatchRun{}, err
	}

	result, err := r.Enricher.MatchTrace(ctx, request)
	if err != nil {
		return RouteMatchRun{}, err
	}

	matchedAt := time.Now().UTC()
	if r.Now != nil {
		matchedAt = r.Now().UTC()
	}
	trace, err := routing.EnrichedTraceFromMatchTraceResult(result, len(points), matchedAt)
	if err != nil {
		return RouteMatchRun{}, err
	}

	pointIDs := pointIDsFromPoints(points)
	runID, err := r.Store.SaveRouteMatchRun(ctx, trace, pointIDs, options.Start, options.End)
	if err != nil {
		return RouteMatchRun{}, err
	}
	return r.Store.GetRouteMatchRun(ctx, runID)
}

func pointIDsFromPoints(points []Point) []int64 {
	ids := make([]int64, 0, len(points))
	for _, point := range points {
		ids = append(ids, point.ID)
	}
	return ids
}
