package storage

import (
	"fmt"
	"os"
)

// OpenOutputFile refuses to overwrite existing files unless force is true.
func OpenOutputFile(path string, force bool) (*os.File, error) {
	if path == "-" {
		return os.Stdout, nil
	}
	flags := os.O_WRONLY | os.O_CREATE
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("output file %q already exists; use --force to overwrite", path)
		}
		return nil, err
	}
	return f, nil
}
