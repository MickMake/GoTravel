package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/MickMake/GoTravel/storage"
)

func runDB(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("db requires a subcommand: init, verify, export, or import")
	}

	switch args[0] {
	case "init":
		return runDBInit(args[1:])
	case "verify":
		return runDBVerify(args[1:])
	case "export":
		return runDBExport(args[1:])
	case "import":
		return runDBImport(args[1:])
	default:
		return fmt.Errorf("unknown db subcommand %q", args[0])
	}
}

func runDBInit(args []string) error {
	fs := flag.NewFlagSet("db init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	force := fs.Bool("force", false, "replace any existing database")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("db init does not accept positional arguments")
	}

	if err := storage.InitDatabase(*dbPath, *force); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "database initialised: %s\n", *dbPath)
	return nil
}

func runDBExport(args []string) error {
	fs := flag.NewFlagSet("db export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	force := fs.Bool("force", false, "overwrite output database file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("db export requires exactly one output filename")
	}

	if err := storage.ExportDatabase(*dbPath, fs.Arg(0), *force); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "database exported: %s -> %s\n", *dbPath, fs.Arg(0))
	return nil
}

func runDBImport(args []string) error {
	fs := flag.NewFlagSet("db import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	force := fs.Bool("force", false, "overwrite existing database")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("db import requires exactly one input filename")
	}

	if err := storage.ImportDatabase(fs.Arg(0), *dbPath, *force); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "database imported: %s -> %s\n", fs.Arg(0), *dbPath)
	return nil
}
