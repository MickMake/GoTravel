package routing

import "time"

// Capabilities describes which common operations a provider supports.
type Capabilities struct {
	Route      bool
	MatchTrace bool
	Snap       bool
	Matrix     bool
}

// Coordinate is a provider-neutral lon/lat point.
type Coordinate struct {
	Lat float64
	Lng float64
}

// TracePoint is a timestamped coordinate for map-matching operations.
type TracePoint struct {
	Coordinate
	Time     time.Time
	Accuracy *float64
	Radius   *float64
}

type RouteRequest struct {
	Profile string
	Start   Coordinate
	End     Coordinate
}

type RouteResult struct {
	Provider        string
	Profile         string
	Status          string
	Geometry        string
	GeometryFormat  string
	DistanceMeters  float64
	DurationSeconds float64
	Warnings        []string
	RawResponse     []byte
}

type MatchTraceRequest struct {
	Profile string
	Points  []TracePoint
}

type MatchTraceResult struct {
	Provider        string
	Profile         string
	Status          string
	Geometry        string
	GeometryFormat  string
	DistanceMeters  float64
	DurationSeconds float64
	Confidence      *float64
	Warnings        []string
	RawResponse     []byte
}

type SnapRequest struct {
	Profile     string
	Coordinates []Coordinate
}

type SnapResult struct {
	Provider    string
	Profile     string
	Status      string
	Points      []Coordinate
	Warnings    []string
	RawResponse []byte
}

type MatrixRequest struct {
	Profile      string
	Sources      []Coordinate
	Destinations []Coordinate
}

type MatrixResult struct {
	Provider       string
	Profile        string
	Status         string
	DurationMatrix [][]float64
	DistanceMatrix [][]float64
	Warnings       []string
	RawResponse    []byte
}
