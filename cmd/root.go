package cmd

import (
	"fmt"
	"os"
)

func Execute() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "db":
		return runDB(args[1:])
	case "import":
		return runImport(args[1:])
	case "export":
		return runExport(args[1:])
	case "route-match":
		return runRouteMatch(args[1:])
	case "trips":
		return runTrips(args[1:])
	case "help", "--help", "-h":
		return usage()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() error {
	fmt.Fprint(os.Stderr, `GoTravel 0.5

Usage:
  GoTravel db init [--db gotravel.sqlite] [--force]
  GoTravel db verify [--db gotravel.sqlite]
  GoTravel db export [--db gotravel.sqlite] [--force] file
  GoTravel db import [--db gotravel.sqlite] [--force] file
  GoTravel import [--db gotravel.sqlite] [--force] gator|google input.csv ...
  GoTravel import [--db gotravel.sqlite] [--force] gator|google -
  GoTravel export gator|google|gpx output [--db gotravel.sqlite] [--force] [--start value] [--stop value]
  GoTravel route-match run [--db gotravel.sqlite] [--provider noop|osrm] [--profile value] [--osrm-base-url url] [--from value] [--to value] [--radius meters]
  GoTravel route-match inspect [--db gotravel.sqlite] run-id
  GoTravel route-match export [--db gotravel.sqlite] [--force] geojson|gpx run-id output
  GoTravel trips segment [--db gotravel.sqlite] [--gap-minutes 30] [--force]
  GoTravel trips list [--db gotravel.sqlite]
  GoTravel trips inspect [--db gotravel.sqlite] trip-id
`)
	return nil
}
