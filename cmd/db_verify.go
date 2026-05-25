package cmd

import (
	"flag"
	"fmt"
	"io"

	"github.com/MickMake/GoTravel/storage"
)

func runDBVerify(args []string) error {
	fs := flag.NewFlagSet("db verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("db verify does not accept positional arguments")
	}
	return storage.ValidateDatabase(*dbPath)
}
