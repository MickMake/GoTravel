package cmd

import (
	"context"
	"encoding/json"
	"encoding/xml"
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

type routeMatchGPXDocument struct {
	XMLName xml.Name               `xml:"gpx"`
	Version string                 `xml:"version,attr"`
	Creator string                 `xml:"creator,attr"`
	XMLNS   string                 `xml:"xmlns,attr"`
	Track   routeMatchGPXTrack     `xml:"trk"`
}

type routeMatchGPXTrack struct {
	Name    string                  `xml:"name,omitempty"`
	Segment routeMatchGPXSegment   `xml:"trkseg"`
}

type routeMatchGPXSegment struct {
	Points []routeMatchGPXPoint     `xml:"trkpt"`
}

type routeMatchGPXPoint struct {
	Lat string `xml:"lat,attr"`
	Lon string `xml:"lon,attr"`
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
  GoTravel route-match export [--db gotravel.sqlite] [--force] <geojson|gpx> <run-id> <output.geojson|output.gpx|->
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
	if format != "geojson" && format != "gpx" {
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
	switch format {
	case "geojson":
		return writeRouteMatchGeoJSON(out, run)
	case "gpx":
		return writeRouteMatchGPX(out, run)
	default:
		return fmt.Errorf("unsupported route-match export format %q", format)
	}
}

func addRouteMatchCommonFlags(fs *flag.FlagSet) *routeMatchCommonArgs {
	common := &routeMatchCommonArgs{}
	fs.StringVar(&common.dbPath, "db", "gotravel.sqlite", "SQLite database path")
	fs.StringVar(&common.provider, "provider", "noop", "routing provider name")
	fs.StringVar(&common.profile, "profile", "driving", "routing profile")
	fs.StringVar(&common.osrmBaseURL, "osrm-base-url", "OSRM base URL")
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
	geometry, err := routing.RouteGeometryAsGeoJSON(run.Trace.GeometryFormat, run.Trace.Geometry)
	if err != nil {
		return nil, fmt.Errorf("route-match run %d geometry cannot be exported as GeoJSON: %w", run.ID, err)
	}
	return geometry, nil
}

func writeRouteMatchGPX(w io.Writer, run storage.RouteMatchRun) error {
	coordinates, err := routing.RouteGeometryCoordinates(run.Trace.GeometryFormat, run.Trace.Geometry)
	if err != nil {
		return fmt.Errorf("route-match run %d geometry cannot be exported as GPX: %w", run.ID, err)
	}
	if len(coordinates) == 0 {
		return fmt.Errorf("route-match run %d has no coordinates", run.ID)
	}

	doc := routeMatchGPXDocument{
		Version: "1.1",
		Creator: "GoTravel",
		XMLNS:   "http://www.topografix.com/GPX/1/1",
		Track: routeMatchGPXTrack{
			Name:    fmt.Sprintf("GoTravel route match %d", run.ID),
			Segment: routeMatchGPXSegment{Points: make([]routeMatchGPXPoint, 0, len(coordinates))},
		},
	}
	for _, coordinate := range coordinates {
		doc.Track.Segment.Points = append(doc.Track.Segment.Points, routeMatchGPXPoint{
			Lat: fmt.Sprintf("%.7f", coordinate.Lat),
			Lon: fmt.Sprintf("%.7f", coordinate.Lon),
		})
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(doc); err != nil {
		return err
	}
	if err := encoder.Flush(); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}
