package storage

import "fmt"

// CorruptInputError is returned when import data is corrupt and --force was not used.
type CorruptInputError struct {
	SourceFile string
	Line       int
	Reason     string
}

func (e CorruptInputError) Error() string {
	return fmt.Sprintf("corrupt input in %s line %d: %s", e.SourceFile, e.Line, e.Reason)
}
