package routing

import (
	"fmt"
	"time"
)

// EnrichedTrace is GoTravel's provider-neutral model for a matched GPS trace.
type EnrichedTrace struct {
	Provider         string
	Profile          string
	Status           string
	SourcePointCount int
	Geometry         string
	GeometryFormat   string
	DistanceMeters   float64
	DurationSeconds  float64
	Confidence       *float64
	Warnings         []string
	RawResponse      []byte
	MatchedAt        time.Time
}

// EnrichedTraceFromMatchTraceResult creates an internal enriched-trace model from a provider result.
func EnrichedTraceFromMatchTraceResult(result MatchTraceResult, sourcePointCount int, matchedAt time.Time) (EnrichedTrace, error) {
	if sourcePointCount < 2 {
		return EnrichedTrace{}, fmt.Errorf("routing enriched trace: at least two source points are required")
	}
	if matchedAt.IsZero() {
		return EnrichedTrace{}, fmt.Errorf("routing enriched trace: matched timestamp is required")
	}

	enriched := EnrichedTrace{
		Provider:         result.Provider,
		Profile:          result.Profile,
		Status:           result.Status,
		SourcePointCount: sourcePointCount,
		Geometry:         result.Geometry,
		GeometryFormat:   result.GeometryFormat,
		DistanceMeters:   result.DistanceMeters,
		DurationSeconds:  result.DurationSeconds,
		Warnings:         append([]string(nil), result.Warnings...),
		RawResponse:      append([]byte(nil), result.RawResponse...),
		MatchedAt:        matchedAt,
	}
	if result.Confidence != nil {
		confidence := *result.Confidence
		enriched.Confidence = &confidence
	}
	return enriched, nil
}
