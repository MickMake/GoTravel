package cmd

import (
	"flag"
	"fmt"
	"io"

	exporters "github.com/MickMake/GoTravel/export"
	"github.com/MickMake/GoTravel/storage"
)

func runExport(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	force := fs.Bool("force", false, "overwrite output files")
	startRaw := fs.String("start", "", "inclusive start date/time")
	stopRaw := fs.String("stop", "", "inclusive stop date/time")
	format := fs.String("format", "csv", "export format")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) != 1 {
		return fmt.Errorf("export requires exactly one output path or '-'")
	}

	start, err := storage.ParsePartialDateTime(*startRaw, false)
	if err != nil {
		return err
	}
	stop, err := storage.ParsePartialDateTime(*stopRaw, true)
	if err != nil {
		return err
	}

	store, err := storage.Open(*dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	points, err := store.QueryPoints(start, stop)
	if err != nil {
		return err
	}

	exp, err := exporters.New(*format)
	if err != nil {
		return err
	}

	out, err := storage.OpenOutputFile(remaining[0], *force)
	if err != nil {
		return err
	}
	if remaining[0] != "-" {
		defer out.Close()
	}
	return exp.Export(out, points)
}
