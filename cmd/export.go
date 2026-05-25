package cmd

import (
	"fmt"
	"strings"

	exporters "github.com/MickMake/GoTravel/export"
	"github.com/MickMake/GoTravel/storage"
)

type exportArgs struct {
	dbPath      string
	force       bool
	startRaw    string
	stopRaw     string
	positionals []string
}

func runExport(args []string) error {
	parsed, err := parseExportArgs(args)
	if err != nil {
		return err
	}
	if len(parsed.positionals) != 2 {
		return fmt.Errorf("export requires a format and output path: GoTravel export <gator|google> <output.csv|-> [--db path] [--force] [--start value] [--stop value]")
	}

	format := parsed.positionals[0]
	outputPath := parsed.positionals[1]

	start, err := storage.ParsePartialDateTime(parsed.startRaw, false)
	if err != nil {
		return err
	}
	stop, err := storage.ParsePartialDateTime(parsed.stopRaw, true)
	if err != nil {
		return err
	}

	store, err := storage.Open(parsed.dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	points, err := store.QueryPoints(start, stop)
	if err != nil {
		return err
	}

	exp, err := exporters.New(format)
	if err != nil {
		return err
	}

	out, err := storage.OpenOutputFile(outputPath, parsed.force)
	if err != nil {
		return err
	}
	if outputPath != "-" {
		defer out.Close()
	}
	return exp.Export(out, points)
}

func parseExportArgs(args []string) (exportArgs, error) {
	parsed := exportArgs{dbPath: "gotravel.sqlite"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--force":
			parsed.force = true
		case arg == "--db" || arg == "--start" || arg == "--stop":
			if i+1 >= len(args) {
				return parsed, fmt.Errorf("%s requires a value", arg)
			}
			value := args[i+1]
			i++
			switch arg {
			case "--db":
				parsed.dbPath = value
			case "--start":
				parsed.startRaw = value
			case "--stop":
				parsed.stopRaw = value
			}
		case strings.HasPrefix(arg, "--db="):
			parsed.dbPath = strings.TrimPrefix(arg, "--db=")
		case strings.HasPrefix(arg, "--start="):
			parsed.startRaw = strings.TrimPrefix(arg, "--start=")
		case strings.HasPrefix(arg, "--stop="):
			parsed.stopRaw = strings.TrimPrefix(arg, "--stop=")
		case strings.HasPrefix(arg, "--"):
			return parsed, fmt.Errorf("unknown export option %q", arg)
		default:
			parsed.positionals = append(parsed.positionals, arg)
		}
	}
	return parsed, nil
}
