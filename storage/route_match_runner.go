package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MickMake/GoTravel/routing"
)

const (
	routeMatchTraceChunkMaxPoints = 100
	routeMatchTraceChunkOverlap   = 1
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
	cleanedPoints := removeConsecutiveDuplicateCoordinates(points)
	if len(cleanedPoints) < 2 {
		return RouteMatchRun{}, fmt.Errorf("storage route match runner: at least two points are required after trace hygiene")
	}

	chunks := routeMatchTraceChunks(cleanedPoints, routeMatchTraceChunkMaxPoints, routeMatchTraceChunkOverlap)
	if len(chunks) == 0 {
		return RouteMatchRun{}, fmt.Errorf("storage route match runner: no valid route-match chunks")
	}

	results := make([]routing.MatchTraceResult, 0, len(chunks))
	for i, chunk := range chunks {
		request, err := MatchTraceRequestFromPoints(chunk, MatchTraceOptions{Profile: options.Profile, Radius: options.Radius})
		if err != nil {
			return RouteMatchRun{}, fmt.Errorf("storage route match runner: chunk %d/%d request: %w", i+1, len(chunks), err)
		}

		result, err := r.Enricher.MatchTrace(ctx, request)
		if err != nil {
			return RouteMatchRun{}, fmt.Errorf("storage route match runner: chunk %d/%d provider match failed: %w", i+1, len(chunks), err)
		}
		results = append(results, result)
	}

	matchedAt := time.Now().UTC()
	if r.Now != nil {
		matchedAt = r.Now().UTC()
	}
	trace, err := enrichedTraceFromChunkResults(results, len(cleanedPoints), matchedAt)
	if err != nil {
		return RouteMatchRun{}, err
	}

	pointIDs := pointIDsFromPoints(cleanedPoints)
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

func removeConsecutiveDuplicateCoordinates(points []Point) []Point {
	if len(points) == 0 {
		return nil
	}
	cleaned := make([]Point, 0, len(points))
	for _, point := range points {
		if len(cleaned) > 0 {
			previous := cleaned[len(cleaned)-1]
			if previous.Lat == point.Lat && previous.Lng == point.Lng {
				continue
			}
		}
		cleaned = append(cleaned, point)
	}
	return cleaned
}

func routeMatchTraceChunks(points []Point, maxPoints, overlap int) [][]Point {
	if maxPoints < 2 || len(points) < 2 {
		return nil
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= maxPoints {
		overlap = maxPoints - 1
	}

	chunks := make([][]Point, 0)
	for start := 0; start < len(points); {
		end := start + maxPoints
		if end > len(points) {
			end = len(points)
		}
		if end-start >= 2 {
			chunk := append([]Point(nil), points[start:end]...)
			chunks = append(chunks, chunk)
		}
		if end == len(points) {
			break
		}
		start = end - overlap
	}
	return chunks
}

func enrichedTraceFromChunkResults(results []routing.MatchTraceResult, sourcePointCount int, matchedAt time.Time) (routing.EnrichedTrace, error) {
	if len(results) == 0 {
		return routing.EnrichedTrace{}, fmt.Errorf("storage route match runner: no route-match chunk results")
	}
	if len(results) == 1 {
		return routing.EnrichedTraceFromMatchTraceResult(results[0], sourcePointCount, matchedAt)
	}

	coordinates, err := combinedChunkCoordinates(results)
	if err != nil {
		return routing.EnrichedTrace{}, err
	}
	geometry, err := geoJSONLineString(coordinates)
	if err != nil {
		return routing.EnrichedTrace{}, err
	}

	combined := routing.MatchTraceResult{
		Provider:       results[0].Provider,
		Profile:        results[0].Profile,
		Status:         results[0].Status,
		Geometry:       geometry,
		GeometryFormat: "geojson",
		RawResponse:    combinedRawResponses(results),
	}
	for chunkIndex, result := range results {
		combined.DistanceMeters += result.DistanceMeters
		combined.DurationSeconds += result.DurationSeconds
		for _, warning := range result.Warnings {
			combined.Warnings = append(combined.Warnings, fmt.Sprintf("chunk %d: %s", chunkIndex+1, warning))
		}
	}
	if confidence, ok := averageConfidence(results); ok {
		combined.Confidence = &confidence
	}
	return routing.EnrichedTraceFromMatchTraceResult(combined, sourcePointCount, matchedAt)
}

func combinedChunkCoordinates(results []routing.MatchTraceResult) ([]routing.Coordinate, error) {
	combined := make([]routing.Coordinate, 0)
	for i, result := range results {
		coordinates, err := routing.RouteGeometryCoordinates(result.GeometryFormat, result.Geometry)
		if err != nil {
			return nil, fmt.Errorf("storage route match runner: chunk %d geometry: %w", i+1, err)
		}
		if len(coordinates) < 2 {
			return nil, fmt.Errorf("storage route match runner: chunk %d geometry has fewer than two coordinates", i+1)
		}
		for _, coordinate := range coordinates {
			if len(combined) > 0 && sameCoordinate(combined[len(combined)-1], coordinate) {
				continue
			}
			combined = append(combined, coordinate)
		}
	}
	if len(combined) < 2 {
		return nil, fmt.Errorf("storage route match runner: combined geometry has fewer than two coordinates")
	}
	return combined, nil
}

func sameCoordinate(a, b routing.Coordinate) bool {
	return a.Lat == b.Lat && a.Lng == b.Lng
}

func geoJSONLineString(coordinates []routing.Coordinate) (string, error) {
	pairs := make([][]float64, 0, len(coordinates))
	for _, coordinate := range coordinates {
		pairs = append(pairs, []float64{coordinate.Lng, coordinate.Lat})
	}
	payload := map[string]any{
		"type":        "LineString",
		"coordinates": pairs,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func combinedRawResponses(results []routing.MatchTraceResult) []byte {
	type rawChunk struct {
		Chunk       int `json:"chunk"`
		RawResponse any `json:"raw_response"`
	}
	chunks := make([]rawChunk, 0, len(results))
	for i, result := range results {
		var raw any
		if len(result.RawResponse) > 0 && json.Valid(result.RawResponse) {
			if err := json.Unmarshal(result.RawResponse, &raw); err != nil {
				raw = string(result.RawResponse)
			}
		} else {
			raw = string(result.RawResponse)
		}
		chunks = append(chunks, rawChunk{Chunk: i + 1, RawResponse: raw})
	}
	encoded, err := json.Marshal(chunks)
	if err != nil {
		return nil
	}
	return encoded
}

func averageConfidence(results []routing.MatchTraceResult) (float64, bool) {
	var total float64
	for _, result := range results {
		if result.Confidence == nil {
			return 0, false
		}
		total += *result.Confidence
	}
	return total / float64(len(results)), true
}
