package importers

import (
	"fmt"
	"io"

	"github.com/MickMake/GoTravel/storage"
)

type Importer interface {
	Import(r io.Reader, source string) storage.ImportResult
}

func New(format string) (Importer, error) {
	switch format {
	case "gator":
		return Gator{}, nil
	case "google":
		return nil, fmt.Errorf("google importer is reserved but not implemented yet")
	default:
		return nil, fmt.Errorf("unknown import format %q", format)
	}
}
