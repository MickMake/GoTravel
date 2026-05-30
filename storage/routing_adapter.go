package storage

import (
	"fmt"
	"math"

	"github.com/MickMake/GoTravel/routing"
)

// MatchTraceOptions controls how staged points are converted into a routing request.
type MatchTraceOptions struct {
	Profile string
	Radius  *float64
}

// MatchTraceRequestFromPoints converts staged points into a provider-neutral routing match request.
func MatchTraceRequestFromPoints(points []Point, options MatchTraceOptions) (routing.MatchTraceRequest, error) {
	if len(points) < 2 {
		return routing.MatchTraceRequest{}, fmt.Errorf("storage routing adapter: at least two points are required")
	}

	tracePoints := make([]routing.TracePoint, 0, len(points))
	for i, point := range points {
		if point.DT.IsZero() {
			return routing.MatchTraceRequest{}, fmt.Errorf("storage routing adapter: point %d has zero timestamp", i)
		}
		if err := validateCoordinate(i, point.Lat, point.Lng); err != nil {
			return routing.MatchTraceRequest{}, err
		}

		tracePoint := routing.TracePoint{
			Coordinate: routing.Coordinate{Lat: point.Lat, Lng: point.Lng},
			Time:       point.DT,
		}
		if options.Radius != nil {
			radius := *options.Radius
			tracePoint.Radius = &radius
		}
		tracePoints = append(tracePoints, tracePoint)
	}

	return routing.MatchTraceRequest{Profile: options.Profile, Points: tracePoints}, nil
}

func validateCoordinate(index int, lat, lng float64) error {
	if !isFinite(lat) {
		return fmt.Errorf("storage routing adapter: point %d latitude %v is not finite", index, lat)
	}
	if lat < -90 || lat > 90 {
		return fmt.Errorf("storage routing adapter: point %d latitude %v outside valid range", index, lat)
	}
	if !isFinite(lng) {
		return fmt.Errorf("storage routing adapter: point %d longitude %v is not finite", index, lng)
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("storage routing adapter: point %d longitude %v outside valid range", index, lng)
	}
	return nil
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
