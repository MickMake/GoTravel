package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MickMake/GoTravel/routing"
	"github.com/MickMake/GoTravel/routing/providers"
	"github.com/MickMake/GoTravel/storage"
)

type routeMatchCommonArgs struct {
	dbPath      string
	provider    string
	profile     string
	osrmBaseURL string
}

func runRouteMatch(args []string) error {
	if len(args) == 0 {
		return routeMatchUsage()
	}
	switch args[0] {
	case "run":
		return runRouteMatchRun(args[1:])
	case "inspect", "show":
		return runRouteMatchInspect(args[1:])
	case "export":
		return runRouteMatchExport(args[1:])
	case "help", "--help", "-h":
		return routeMatchUsage()
	default:
		return fmt.Errorf("unknown route-match command %q", args[0])
	}
}

func routeMatchUsage() error {
	fmt.Fprint(os.Stderr, `GoTravel route-match

Usage:
  GoTravel route-match run [--db gotravel.sqlite] [--provider noop|osrm] [--profile value] [--osrm-base-url url] [--from value] [--to value] [--radius meters]
  GoTravel route-match inspect [--db gotravel.sqlite] <run-id>
  GoTravel route-match export [--db gotravel.sqlite] [--force] geojson <run-id> <output.geojson|->
`)
	return nil
}

func runRouteMatchRun(args []string) error {
	fs := flag.NewFlagSet("route-match run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addRouteMatchCommonFlags(fs)
	fromRaw := fs.String("from", "", "inclusive source point start filter")
	toRaw := fs.String("to", "", "inclusive source point end filter")
	radiusRaw := fs.String("radius", "", "optional per-point route-match radius in metres")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("route-match run does not accept positional arguments")
	}

	from, err := storage.ParsePartialDateTime(*fromRaw, false)
	if err != nil {
		return err
	}
	to, err := storage.ParsePartialDateTime(*toRaw, true)
	if err != nil {
		return err
	}
	radius, err := parseOptionalFloat(*radiusRaw)
	if err != nil {
		return err
	}

	store, err := storage.Open(common.dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	enricher, err := buildRouteMatchEnricher(common)
	if err != nil {
		return err
	}

	runner := storage.RouteMatchRunner{Store: store, Enricher: enricher}
	run, err := runner.RunMatchTrace(context.Background(), storage.RouteMatchRunOptions{
		Profile: common.profile,
		Radius:  radius,
		Start:   from,
		End:     to,
	})
	if err != nil {
		return err
	}
	printRouteMatchSummary(os.Stdout, run)
	return nil
}

func runRouteMatchInspect(args []string) error {
	fs := flag.NewFlagSet("route-match inspect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("route-match inspect requires a run ID")
	}
	runID, err := parseRunID(fs.Arg(0))
	if err != nil {
		return err
	}
	store, err := storage.Open(*dbPath)
	if err != nil {
		return err
	}
	defer store.Close()
	run, err := store.GetRouteMatchRun(context.Background(), runID)
	if err != nil {
		return err
	}
	printRouteMatchSummary(os.Stdout, run)
	fmt.Fprintf(os.Stdout, "linked_point_count=%d\n", len(run.PointIDs))
	fmt.Fprintf(os.Stdout, "matched_at=%s\n", formatRouteMatchTime(run.Trace.MatchedAt))
	fmt.Fprintf(os.Stdout, "created_at=%s\n", formatRouteMatchTime(run.CreatedAt))
	if run.SourceFilterStart != nil {
		fmt.Fprintf(os.Stdout, "source_filter_start=%s\n", formatRouteMatchTime(*run.SourceFilterStart))
	}
	if run.SourceFilterEnd != nil {
		fmt.Fprintf(os.Stdout, "source_filter_end=%s\n", formatRouteMatchTime(*run.SourceFilterEnd))
	}
	if len(run.Trace.Warnings) > 0 {
		fmt.Fprintf(os.Stdout, "warnings=%s\n", strings.Join(run.Trace.Warnings, "; "))
	}
	return nil
}

func runRouteMatchExport(args []string) error {
	fs := flag.NewFlagSet("route-match export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", "gotravel.sqlite", "SQLite database path")
	force := fs.Bool("force", false, "overwrite output file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 3 {
		return fmt.Errorf("route-match export requires format, run ID, and output path")
	}
	format := strings.ToLower(fs.Arg(0))
	if format != "geojson" {
		return fmt.Errorf("unsupported route-match export format %q", format)
	}
	runID, err := parseRunID(fs.Arg(1))
	if err != nil {
		return err
	}
	outputPath := fs.Arg(2)

	store, err := storage.Open(*dbPath)
	if err != nil {
		return err
	}
	defer store.Close()
	run, err := store.GetRouteMatchRun(context.Background(), runID)
	if err != nil {
		return err
	}

	out, err := storage.OpenOutputFile(outputPath, *force)
	if err != nil {
		return err
	}
	if outputPath != "-" {
		defer out.Close()
	}
	return writeRouteMatchGeoJSON(out, run)
}

func addRouteMatchCommonFlags(fs *flag.FlagSet) *routeMatchCommonArgs {
	common := &routeMatchCommonArgs{}
	fs.StringVar(&common.dbPath, "db", "gotravel.sqlite", "SQLite database path")
	fs.StringVar(&common.provider, "provider", "noop", "routing provider name")
	fs.StringVar(&common.profile, "profile", "driving", "routing profile")
	fs.StringVar(&common.osrmBaseURL, "osrm-base-url", "", "OSRM base URL")
	return common
}

func buildRouteMatchEnricher(common *routeMatchCommonArgs) (*routing.Enricher, error) {
	provider, err := providers.New(providers.Config{
		Name: common.provider,
		OSRM: providers.OSRMConfig{
			BaseURL: common.osrmBaseURL,
			Profile: common.profile,
		},
	})
	if err != nil {
		return nil, err
	}
	service, err := routing.NewService(provider)
	if err != nil {
		return nil, err
	}
	return routing.NewEnricher(service)
}

func parseOptionalFloat(value string) (*float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid radius %q", value)
	}
	if parsed < 0 {
		return nil, fmt.Errorf("radius must be non-negative")
	}
	return &parsed, nil
}

func parseRunID(value string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid route-match run ID %q", value)
	}
	return id, nil
}

func printRouteMatchSummary(w io.Writer, run storage.RouteMatchRun) {
	fmt.Fprintf(w, "route_match_run_id=%d\n", run.ID)
	fmt.Fprintf(w, "provider=%s\n", run.Trace.Provider)
	fmt.Fprintf(w, "profile=%s\n", run.Trace.Profile)
	fmt.Fprintf(w, "status=%s\n", run.Trace.Status)
	fmt.Fprintf(w, "source_point_count=%d\n", run.Trace.SourcePointCount)
	fmt.Fprintf(w, "distance_meters=%.3f\n", run.Trace.DistanceMeters)
	fmt.Fprintf(w, "duration_seconds=%.3f\n", run.Trace.DurationSeconds)
	fmt.Fprintf(w, "geometry_format=%s\n", run.Trace.GeometryFormat)
}

func formatRouteMatchTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func writeRouteMatchGeoJSON(w io.Writer, run storage.RouteMatchRun) error {
	geometry, err := routeMatchGeoJSONGeometry(run)
	if err != nil {
		return err
	}
	feature := map[string]any{
		"type":     "Feature",
		"geometry": geometry,
		"properties": map[string]any{
			"route_match_run_id": run.ID,
			"provider":           run.Trace.Provider,
			"profile":            run.Trace.Profile,
			"status":             run.Trace.Status,
			"source_point_count": run.Trace.SourcePointCount,
			"distance_meters":    run.Trace.DistanceMeters,
			"duration_seconds":   run.Trace.DurationSeconds,
			"geometry_format":    run.Trace.GeometryFormat,
			"matched_at":         formatRouteMatchTime(run.Trace.MatchedAt),
			"created_at":         formatRouteMatchTime(run.CreatedAt),
		},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(feature)
}

func routeMatchGeoJSONGeometry(run storage.RouteMatchRun) (any, error) {
	format := strings.ToLower(strings.TrimSpace(run.Trace.GeometryFormat))
	geometry := strings.TrimSpace(run.Trace.Geometry)
	if geometry == "" {
		return nil, fmt.Errorf("route-match run %d has no geometry", run.ID)
	}
	if strings.Contains(format, "geojson") || strings.HasPrefix(geometry, "{") {
		var decoded any
		if err := json.Unmarshal([]byte(geometry), &decoded); err != nil {
			return nil, fmt.Errorf("route-match run %d geometry is not valid GeoJSON: %w", run.ID, err)
		}
		return decoded, nil
	}
	return nil, fmt.Errorf("route-match run %d geometry format %q cannot be exported as GeoJSON yet", run.ID, run.Trace.GeometryFormat)
}
