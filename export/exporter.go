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
	case "csv", "staged-csv", "":
		return CSV{}, nil
	case "gpx":
		return nil, fmt.Errorf("gpx export is reserved but not implemented yet")
	default:
		return nil, fmt.Errorf("unknown export format %q", format)
	}
}
