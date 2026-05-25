package exporters

import (
	"fmt"
	"io"

	"github.com/MickMake/GoTravel/storage"
)

type Exporter interface {
	Export(w io.Writer, points []storage.Point) error
}

func New(format string) (Exporter, error) {
	switch format {
	case "gator", "csv", "staged-csv":
		return CSV{}, nil
	case "gpx":
		return GPX{}, nil
	case "google":
		return nil, fmt.Errorf("google export is reserved but not implemented yet")
	case "":
		return nil, fmt.Errorf("export format is required: gator, google, or gpx")
	default:
		return nil, fmt.Errorf("unknown export format %q", format)
	}
}
