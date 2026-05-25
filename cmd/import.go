package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"

	importers "github.com/MickMake/GoTravel/import"
	"github.com/MickMake/GoTravel/storage"
)

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	force := fs.Bool("force", false, "skip corrupt rows and keep valid rows")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) < 2 {
		return fmt.Errorf("import requires a format and at least one input")
	}

	format := remaining[0]
	inputs := remaining[1:]
	if len(inputs) > 1 {
		for _, input := range inputs {
			if input == "-" {
				return fmt.Errorf("stdin import '-' cannot be combined with other files")
			}
		}
	}

	imp, err := importers.New(format)
	if err != nil {
		return err
	}
	store, err := storage.Open(*dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	totalImported := 0
	totalSkipped := 0
	for _, input := range inputs {
		var r io.Reader
		source := input
		var f *os.File
		if input == "-" {
			r = os.Stdin
			source = "<stdin>"
		} else {
			f, err = os.Open(input)
			if err != nil {
				return err
			}
			r = f
		}

		result := imp.Import(r, source)
		if f != nil {
			_ = f.Close()
		}

		imported, skipped, err := store.SaveImportResult(result, *force)
		if err != nil {
			return err
		}
		totalImported += imported
		totalSkipped += skipped
		fmt.Fprintf(os.Stderr, "%s: imported=%d skipped=%d errors=%d\n", source, imported, skipped, len(result.Errors))
	}

	fmt.Fprintf(os.Stderr, "total: imported=%d skipped=%d\n", totalImported, totalSkipped)
	return nil
}
