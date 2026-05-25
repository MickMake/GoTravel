package routing

import (
	"context"

	"github.com/MickMake/GoTravel/storage"
)

type Route struct {
	DistanceMeters  float64
	DurationSeconds float64
	Geometry        string
}

type Router interface {
	Route(ctx context.Context, from storage.Point, to storage.Point) (*Route, error)
}
