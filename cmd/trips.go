package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MickMake/GoTravel/storage"
)

func runTrips(args []string) error {
	if len(args) == 0 {
		return tripsUsage()
	}
	switch args[0] {
	case "segment":
		return runTripsSegment(args[1:])
	case "list":
		return runTripsList(args[1:])
	case "inspect", "show":
		return runTripsInspect(args[1:])
	case "help", "--help", "-h":
		return tripsUsage()
	default:
		return fmt.Errorf("unknown trips command %q", args[0])
	}
}

func tripsUsage() error {
	fmt.Fprint(os.Stderr, `GoTravel trips

Usage:
  GoTravel trips segment [--db gotravel.sqlite] [--gap-minutes 30] [--force]
  GoTravel trips list [--db gotravel.sqlite]
  GoTravel trips inspect [--db gotravel.sqlite] <trip-id>
`)
	return nil
}

func runTripsSegment(args []string) error {
	fs := flag.NewFlagSet("trips segment", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	gapMinutes := fs.Int("gap-minutes", storage.DefaultTripGapMinutes, "new-trip gap threshold in minutes")
	force := fs.Bool("force", false, "delete and rebuild existing generated trips")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("trips segment does not accept positional arguments")
	}
	if *gapMinutes <= 0 {
		return fmt.Errorf("gap-minutes must be positive")
	}

	store, err := storage.Open(*dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	trips, err := store.SegmentTrips(context.Background(), storage.SegmentTripsOptions{
		Gap:   time.Duration(*gapMinutes) * time.Minute,
		Force: *force,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "trips_segmented=%d\n", len(trips))
	fmt.Fprintf(os.Stdout, "gap_minutes=%d\n", *gapMinutes)
	return nil
}

func runTripsList(args []string) error {
	fs := flag.NewFlagSet("trips list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("trips list does not accept positional arguments")
	}

	store, err := storage.Open(*dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	trips, err := store.ListTrips(context.Background())
	if err != nil {
		return err
	}
	for _, trip := range trips {
		printTripSummary(os.Stdout, trip)
	}
	return nil
}

func runTripsInspect(args []string) error {
	fs := flag.NewFlagSet("trips inspect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("trips inspect requires a trip ID")
	}
	tripID, err := parseTripID(fs.Arg(0))
	if err != nil {
		return err
	}

	store, err := storage.Open(*dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	trip, err := store.GetTrip(context.Background(), tripID)
	if err != nil {
		return err
	}
	printTripSummary(os.Stdout, trip)
	fmt.Fprintf(os.Stdout, "linked_point_count=%d\n", len(trip.PointIDs))
	fmt.Fprintf(os.Stdout, "point_ids=%s\n", formatTripPointIDs(trip.PointIDs))
	fmt.Fprintf(os.Stdout, "created_at=%s\n", formatTripTime(trip.CreatedAt))
	return nil
}

func parseTripID(value string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid trip ID %q", value)
	}
	return id, nil
}

func printTripSummary(w io.Writer, trip storage.Trip) {
	fmt.Fprintf(w, "trip_id=%d\n", trip.ID)
	fmt.Fprintf(w, "start_time=%s\n", formatTripTime(trip.StartTime))
	fmt.Fprintf(w, "end_time=%s\n", formatTripTime(trip.EndTime))
	fmt.Fprintf(w, "source_point_count=%d\n", trip.SourcePointCount)
	fmt.Fprintf(w, "first_point_id=%d\n", trip.FirstPointID)
	fmt.Fprintf(w, "last_point_id=%d\n", trip.LastPointID)
	fmt.Fprintf(w, "duration_seconds=%d\n", trip.DurationSeconds)
	fmt.Fprintf(w, "gap_seconds=%d\n", trip.GapSeconds)
}

func formatTripTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func formatTripPointIDs(ids []int64) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	return strings.Join(parts, ",")
}
