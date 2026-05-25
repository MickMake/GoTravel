package storage

import "time"

// Point is the canonical internal GPS point model used by GoTravel.
type Point struct {
	ID         int64
	DT         time.Time
	Lat        float64
	Lng        float64
	Altitude   float64
	Angle      float64
	Speed      float64
	Params     string
	Format     string
	SourceFile string
	SourceLine int
	ImportedAt time.Time
	PointHash  string
}

// ImportError records a corrupt or unusable input row.
type ImportError struct {
	SourceFile string
	SourceLine int
	Format     string
	RawRow     string
	Error      string
}

// ImportResult is returned by an importer after reading one source.
type ImportResult struct {
	Format     string
	SourceFile string
	RowsSeen   int
	Points     []Point
	Errors     []ImportError
}
