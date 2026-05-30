package valhalla

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/MickMake/GoTravel/routing"
)

const (
	Name                  = "valhalla"
	defaultBaseURL        = "http://127.0.0.1:8002"
	defaultProfile        = "auto"
	defaultGeometryFormat = "polyline6"
	distanceMultiplier    = 1000.0
)

// Config contains the minimal settings needed to talk to a Valhalla HTTP server.
type Config struct {
	BaseURL    string
	Profile    string
	HTTPClient *http.Client
}

type Provider struct {
	baseURL    string
	profile    string
	httpClient *http.Client
}

func New() *Provider {
	return NewWithConfig(Config{})
}

func NewWithConfig(cfg Config) *Provider {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	profile := cfg.Profile
	if profile == "" {
		profile = defaultProfile
	}

	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	return &Provider{baseURL: baseURL, profile: profile, httpClient: client}
}

func (p *Provider) Name() string { return Name }

func (p *Provider) Health(ctx context.Context) error {
	_, err := p.post(ctx, "locate", map[string]any{
		"costing":   p.profileFor(""),
		"locations": []valhallaLocation{{Lat: 0, Lon: 0}},
	})
	if err != nil {
		return fmt.Errorf("%w: %v", routing.ErrProviderUnavailable, err)
	}
	return nil
}

func (p *Provider) Capabilities(ctx context.Context) routing.Capabilities {
	return routing.Capabilities{Route: true, MatchTrace: true, Snap: true, Matrix: true}
}

func (p *Provider) Route(ctx context.Context, req routing.RouteRequest) (routing.RouteResult, error) {
	profile := p.profileFor(req.Profile)
	result := routing.RouteResult{Provider: Name, Profile: profile, GeometryFormat: defaultGeometryFormat}
	if req.Start == req.End {
		return result, fmt.Errorf("valhalla route: start and end coordinates must differ")
	}

	raw, err := p.post(ctx, "route", map[string]any{
		"costing":            profile,
		"shape_format":       defaultGeometryFormat,
		"locations":          []valhallaLocation{fromCoordinate(req.Start), fromCoordinate(req.End)},
		"directions_options": map[string]string{"units": "kilometers"},
	})
	result.RawResponse = raw
	if err != nil {
		return result, err
	}

	var parsed tripResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("valhalla route: decode response: %w", err)
	}
	applyTripStatus(&result.Status, &result.Warnings, parsed.Trip.Status, parsed.Trip.StatusMessage)
	if parsed.Trip.Status != 0 {
		return result, statusError("route", parsed.Trip.Status, parsed.Trip.StatusMessage)
	}
	shape, err := tripShape(parsed.Trip)
	if err != nil {
		return result, fmt.Errorf("valhalla route: %w", err)
	}
	result.Geometry = shape
	result.DistanceMeters = parsed.Trip.Summary.Length * distanceMultiplier
	result.DurationSeconds = parsed.Trip.Summary.Time
	return result, nil
}

func (p *Provider) MatchTrace(ctx context.Context, req routing.MatchTraceRequest) (routing.MatchTraceResult, error) {
	profile := p.profileFor(req.Profile)
	result := routing.MatchTraceResult{Provider: Name, Profile: profile, GeometryFormat: defaultGeometryFormat}
	if len(req.Points) < 2 {
		return result, fmt.Errorf("valhalla match: at least two trace points are required")
	}

	shape := make([]valhallaTracePoint, 0, len(req.Points))
	for _, point := range req.Points {
		shape = append(shape, fromTracePoint(point))
	}
	raw, err := p.post(ctx, "trace_route", map[string]any{
		"costing":            profile,
		"shape_format":       defaultGeometryFormat,
		"shape_match":        "map_snap",
		"shape":              shape,
		"directions_options": map[string]string{"units": "kilometers"},
	})
	result.RawResponse = raw
	if err != nil {
		return result, err
	}

	var parsed tripResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("valhalla match: decode response: %w", err)
	}
	applyTripStatus(&result.Status, &result.Warnings, parsed.Trip.Status, parsed.Trip.StatusMessage)
	if parsed.Trip.Status != 0 {
		return result, statusError("match", parsed.Trip.Status, parsed.Trip.StatusMessage)
	}
	shapeResult, err := tripShape(parsed.Trip)
	if err != nil {
		return result, fmt.Errorf("valhalla match: %w", err)
	}
	result.Geometry = shapeResult
	result.DistanceMeters = parsed.Trip.Summary.Length * distanceMultiplier
	result.DurationSeconds = parsed.Trip.Summary.Time
	result.Confidence = parsed.Trip.Confidence
	return result, nil
}

func (p *Provider) Snap(ctx context.Context, req routing.SnapRequest) (routing.SnapResult, error) {
	profile := p.profileFor(req.Profile)
	result := routing.SnapResult{Provider: Name, Profile: profile, Status: "Ok"}
	if len(req.Coordinates) == 0 {
		return result, fmt.Errorf("valhalla snap: at least one coordinate is required")
	}

	raw, err := p.post(ctx, "locate", map[string]any{
		"costing":   profile,
		"locations": coordinates(req.Coordinates),
	})
	result.RawResponse = raw
	if err != nil {
		return result, err
	}

	points, err := parseLocatePoints(raw, len(req.Coordinates))
	if err != nil {
		return result, err
	}
	result.Points = points
	return result, nil
}

func (p *Provider) Matrix(ctx context.Context, req routing.MatrixRequest) (routing.MatrixResult, error) {
	profile := p.profileFor(req.Profile)
	result := routing.MatrixResult{Provider: Name, Profile: profile, Status: "Ok"}
	if len(req.Sources) == 0 {
		return result, fmt.Errorf("valhalla matrix: at least one source coordinate is required")
	}
	if len(req.Destinations) == 0 {
		return result, fmt.Errorf("valhalla matrix: at least one destination coordinate is required")
	}

	raw, err := p.post(ctx, "sources_to_targets", map[string]any{
		"costing": profile,
		"sources": coordinates(req.Sources),
		"targets": coordinates(req.Destinations),
		"units":   "kilometers",
	})
	result.RawResponse = raw
	if err != nil {
		return result, err
	}

	var parsed matrixResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("valhalla matrix: decode response: %w", err)
	}
	durations, distances, err := matrixValues(parsed.SourcesToTargets, len(req.Sources), len(req.Destinations))
	if err != nil {
		return result, err
	}
	result.DurationMatrix = durations
	result.DistanceMatrix = distances
	return result, nil
}

func (p *Provider) post(ctx context.Context, service string, payload any) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("valhalla %s: nil provider", service)
	}
	if p.baseURL == "" {
		return nil, fmt.Errorf("valhalla %s: base URL is required", service)
	}

	endpoint, err := url.Parse(p.baseURL)
	if err != nil {
		return nil, fmt.Errorf("valhalla %s: invalid base URL: %w", service, err)
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + service

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("valhalla %s: encode request: %w", service, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("valhalla %s: build request: %w", service, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("valhalla %s: request failed: %w", service, err)
	}
	defer resp.Body.Close()

	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return responseBody, fmt.Errorf("valhalla %s: read response: %w", service, readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseBody, fmt.Errorf("valhalla %s: HTTP %d: %s", service, resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}

func (p *Provider) profileFor(profile string) string {
	if profile != "" {
		return profile
	}
	if p != nil && p.profile != "" {
		return p.profile
	}
	return defaultProfile
}

func fromCoordinate(coord routing.Coordinate) valhallaLocation {
	return valhallaLocation{Lat: coord.Lat, Lon: coord.Lng}
}

func coordinates(coords []routing.Coordinate) []valhallaLocation {
	locations := make([]valhallaLocation, 0, len(coords))
	for _, coord := range coords {
		locations = append(locations, fromCoordinate(coord))
	}
	return locations
}

func fromTracePoint(point routing.TracePoint) valhallaTracePoint {
	tracePoint := valhallaTracePoint{Lat: point.Lat, Lon: point.Lng}
	if !point.Time.IsZero() {
		tracePoint.Time = point.Time.Unix()
	}
	if point.Radius != nil {
		tracePoint.SearchRadius = point.Radius
	} else if point.Accuracy != nil {
		tracePoint.Accuracy = point.Accuracy
	}
	return tracePoint
}

func tripShape(trip valhallaTrip) (string, error) {
	if len(trip.Legs) == 0 {
		return "", fmt.Errorf("no legs returned")
	}
	parts := make([]string, 0, len(trip.Legs))
	for _, leg := range trip.Legs {
		if leg.Shape == "" {
			return "", fmt.Errorf("leg shape is empty")
		}
		parts = append(parts, leg.Shape)
	}
	return strings.Join(parts, ""), nil
}

func applyTripStatus(status *string, warnings *[]string, code int, message string) {
	if code == 0 {
		*status = "Ok"
		return
	}
	*status = fmt.Sprintf("%d", code)
	if message != "" {
		*warnings = append(*warnings, message)
	}
}

func statusError(service string, code int, message string) error {
	if message == "" {
		return fmt.Errorf("valhalla %s: status %d", service, code)
	}
	return fmt.Errorf("valhalla %s: status %d: %s", service, code, message)
}

func parseLocatePoints(raw []byte, want int) ([]routing.Coordinate, error) {
	var list []locateEntry
	if err := json.Unmarshal(raw, &list); err == nil && len(list) > 0 {
		return locateListPoints(list, want)
	}

	var wrapped struct {
		Locations []locateEntry `json:"locations"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("valhalla snap: decode response: %w", err)
	}
	return locateListPoints(wrapped.Locations, want)
}

func locateListPoints(list []locateEntry, want int) ([]routing.Coordinate, error) {
	if len(list) != want {
		return nil, fmt.Errorf("valhalla snap: locate returned %d locations, want %d", len(list), want)
	}
	points := make([]routing.Coordinate, 0, len(list))
	for i, entry := range list {
		coord, ok := entry.bestCoordinate()
		if !ok {
			return nil, fmt.Errorf("valhalla snap: no snapped coordinate returned for location %d", i)
		}
		points = append(points, coord)
	}
	return points, nil
}

func (entry locateEntry) bestCoordinate() (routing.Coordinate, bool) {
	for _, candidate := range entry.CorrelatedLatLons {
		if coord, ok := candidate.coordinate(); ok {
			return coord, true
		}
	}
	for _, edge := range entry.Edges {
		if coord, ok := edge.CorrelatedLatLon.coordinate(); ok {
			return coord, true
		}
	}
	for _, node := range entry.Nodes {
		if coord, ok := node.coordinate(); ok {
			return coord, true
		}
	}
	return entry.coordinate()
}

func (entry locateEntry) coordinate() (routing.Coordinate, bool) {
	if coord, ok := entry.ProjectedLatLon.coordinate(); ok {
		return coord, true
	}
	return entry.LatLon.coordinate()
}

func (point locatePoint) coordinate() (routing.Coordinate, bool) {
	lat, lon := firstNonZero(point.Lat, point.Latitude), firstNonZero(point.Lon, point.Longitude)
	if lat == 0 && lon == 0 {
		return routing.Coordinate{}, false
	}
	return routing.Coordinate{Lat: lat, Lng: lon}, true
}

func firstNonZero(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func matrixValues(rows [][]matrixCell, sources, destinations int) ([][]float64, [][]float64, error) {
	if len(rows) != sources {
		return nil, nil, fmt.Errorf("valhalla matrix: matrix has %d rows, want %d", len(rows), sources)
	}
	durations := make([][]float64, sources)
	distances := make([][]float64, sources)
	for i, row := range rows {
		if len(row) != destinations {
			return nil, nil, fmt.Errorf("valhalla matrix: matrix row %d has %d columns, want %d", i, len(row), destinations)
		}
		durations[i] = make([]float64, destinations)
		distances[i] = make([]float64, destinations)
		for j, cell := range row {
			durations[i][j] = cell.Time
			distances[i][j] = cell.Distance * distanceMultiplier
		}
	}
	return durations, distances, nil
}

type valhallaLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type valhallaTracePoint struct {
	Lat          float64  `json:"lat"`
	Lon          float64  `json:"lon"`
	Time         int64    `json:"time,omitempty"`
	Accuracy     *float64 `json:"accuracy,omitempty"`
	SearchRadius *float64 `json:"search_radius,omitempty"`
}

type tripResponse struct {
	Trip valhallaTrip `json:"trip"`
}

type valhallaTrip struct {
	Status        int             `json:"status"`
	StatusMessage string          `json:"status_message"`
	Summary       valhallaSummary `json:"summary"`
	Legs          []valhallaLeg   `json:"legs"`
	Confidence    *float64        `json:"confidence"`
}

type valhallaSummary struct {
	Length float64 `json:"length"`
	Time   float64 `json:"time"`
}

type valhallaLeg struct {
	Shape string `json:"shape"`
}

type locateEntry struct {
	LatLon            locatePoint  `json:"lat_lon"`
	ProjectedLatLon   locatePoint  `json:"projected_lat_lon"`
	CorrelatedLatLons []locatePoint `json:"correlated_lat_lons"`
	Edges             []locateEdge  `json:"edges"`
	Nodes             []locatePoint `json:"nodes"`
}

type locateEdge struct {
	CorrelatedLatLon locatePoint `json:"correlated_lat_lon"`
}

type locatePoint struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type matrixResponse struct {
	SourcesToTargets [][]matrixCell `json:"sources_to_targets"`
}

type matrixCell struct {
	Distance float64 `json:"distance"`
	Time     float64 `json:"time"`
}
