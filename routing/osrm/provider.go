package osrm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MickMake/GoTravel/routing"
)

const (
	Name                  = "osrm"
	defaultBaseURL        = "http://127.0.0.1:5000"
	defaultProfile        = "driving"
	defaultGeometryFormat = "polyline6"
)

// Config contains the minimal settings needed to talk to an OSRM HTTP server.
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

	return &Provider{
		baseURL:    baseURL,
		profile:    profile,
		httpClient: client,
	}
}

func (p *Provider) Name() string { return Name }

func (p *Provider) Health(ctx context.Context) error {
	_, err := p.get(ctx, "nearest", p.profileFor(""), []routing.Coordinate{{Lat: 0, Lng: 0}}, url.Values{"number": []string{"1"}})
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
	query := url.Values{
		"overview":   []string{"full"},
		"geometries": []string{defaultGeometryFormat},
	}
	raw, err := p.get(ctx, "route", profile, []routing.Coordinate{req.Start, req.End}, query)
	result := routing.RouteResult{Provider: Name, Profile: profile, GeometryFormat: defaultGeometryFormat, RawResponse: raw}
	if err != nil {
		return result, err
	}

	var parsed routeResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("osrm route: decode response: %w", err)
	}
	result.Status = parsed.Code
	result.Warnings = warningsFromMessage(parsed.Message)
	if parsed.Code != "Ok" {
		return result, statusError("route", parsed.Code, parsed.Message)
	}
	if len(parsed.Routes) == 0 {
		return result, fmt.Errorf("osrm route: no routes returned")
	}

	route := parsed.Routes[0]
	result.Geometry = route.Geometry
	result.DistanceMeters = route.Distance
	result.DurationSeconds = route.Duration
	return result, nil
}

func (p *Provider) MatchTrace(ctx context.Context, req routing.MatchTraceRequest) (routing.MatchTraceResult, error) {
	profile := p.profileFor(req.Profile)
	coords := make([]routing.Coordinate, 0, len(req.Points))
	for _, point := range req.Points {
		coords = append(coords, point.Coordinate)
	}
	query := url.Values{
		"overview":   []string{"full"},
		"geometries": []string{defaultGeometryFormat},
	}
	if timestamps, ok := traceTimestamps(req.Points); ok {
		query.Set("timestamps", timestamps)
	}
	if radiuses := traceRadiuses(req.Points); radiuses != "" {
		query.Set("radiuses", radiuses)
	}

	raw, err := p.get(ctx, "match", profile, coords, query)
	result := routing.MatchTraceResult{Provider: Name, Profile: profile, GeometryFormat: defaultGeometryFormat, RawResponse: raw}
	if err != nil {
		return result, err
	}

	var parsed matchResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("osrm match: decode response: %w", err)
	}
	result.Status = parsed.Code
	result.Warnings = warningsFromMessage(parsed.Message)
	if parsed.Code != "Ok" {
		return result, statusError("match", parsed.Code, parsed.Message)
	}
	if len(parsed.Matchings) == 0 {
		return result, fmt.Errorf("osrm match: no matchings returned")
	}

	matching := parsed.Matchings[0]
	result.Geometry = matching.Geometry
	result.DistanceMeters = matching.Distance
	result.DurationSeconds = matching.Duration
	result.Confidence = matching.Confidence
	return result, nil
}

func (p *Provider) Snap(ctx context.Context, req routing.SnapRequest) (routing.SnapResult, error) {
	profile := p.profileFor(req.Profile)
	result := routing.SnapResult{Provider: Name, Profile: profile, Status: "Ok"}
	if len(req.Coordinates) == 0 {
		return result, fmt.Errorf("osrm snap: at least one coordinate is required")
	}

	rawResponses := make([][]byte, 0, len(req.Coordinates))
	for _, coord := range req.Coordinates {
		raw, err := p.get(ctx, "nearest", profile, []routing.Coordinate{coord}, url.Values{"number": []string{"1"}})
		if err != nil {
			result.RawResponse = aggregateRaw(rawResponses)
			return result, err
		}
		rawResponses = append(rawResponses, raw)

		var parsed nearestResponse
		if err := json.Unmarshal(raw, &parsed); err != nil {
			result.RawResponse = aggregateRaw(rawResponses)
			return result, fmt.Errorf("osrm snap: decode response: %w", err)
		}
		if parsed.Code != "Ok" {
			result.Status = parsed.Code
			result.Warnings = append(result.Warnings, warningsFromMessage(parsed.Message)...)
			result.RawResponse = aggregateRaw(rawResponses)
			return result, statusError("nearest", parsed.Code, parsed.Message)
		}
		if len(parsed.Waypoints) == 0 || len(parsed.Waypoints[0].Location) < 2 {
			result.RawResponse = aggregateRaw(rawResponses)
			return result, fmt.Errorf("osrm snap: no waypoint returned")
		}
		result.Points = append(result.Points, routing.Coordinate{Lng: parsed.Waypoints[0].Location[0], Lat: parsed.Waypoints[0].Location[1]})
	}

	result.RawResponse = aggregateRaw(rawResponses)
	return result, nil
}

func (p *Provider) Matrix(ctx context.Context, req routing.MatrixRequest) (routing.MatrixResult, error) {
	profile := p.profileFor(req.Profile)
	coords := append([]routing.Coordinate{}, req.Sources...)
	coords = append(coords, req.Destinations...)
	query := url.Values{
		"annotations":  []string{"duration,distance"},
		"sources":      []string{indexList(0, len(req.Sources))},
		"destinations": []string{indexList(len(req.Sources), len(req.Destinations))},
	}
	raw, err := p.get(ctx, "table", profile, coords, query)
	result := routing.MatrixResult{Provider: Name, Profile: profile, RawResponse: raw}
	if err != nil {
		return result, err
	}

	var parsed tableResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("osrm matrix: decode response: %w", err)
	}
	result.Status = parsed.Code
	result.Warnings = warningsFromMessage(parsed.Message)
	if parsed.Code != "Ok" {
		return result, statusError("table", parsed.Code, parsed.Message)
	}

	durations, err := matrixValues("duration", parsed.Durations)
	if err != nil {
		return result, err
	}
	distances, err := matrixValues("distance", parsed.Distances)
	if err != nil {
		return result, err
	}
	result.DurationMatrix = durations
	result.DistanceMatrix = distances
	return result, nil
}

func (p *Provider) get(ctx context.Context, service, profile string, coords []routing.Coordinate, query url.Values) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("osrm %s: nil provider", service)
	}
	if p.baseURL == "" {
		return nil, fmt.Errorf("osrm %s: base URL is required", service)
	}
	if len(coords) == 0 {
		return nil, fmt.Errorf("osrm %s: at least one coordinate is required", service)
	}

	endpoint, err := url.Parse(p.baseURL)
	if err != nil {
		return nil, fmt.Errorf("osrm %s: invalid base URL: %w", service, err)
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + service + "/v1/" + url.PathEscape(profile) + "/" + coordinateList(coords)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("osrm %s: build request: %w", service, err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osrm %s: request failed: %w", service, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return body, fmt.Errorf("osrm %s: read response: %w", service, readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, fmt.Errorf("osrm %s: HTTP %d: %s", service, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
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

func coordinateList(coords []routing.Coordinate) string {
	parts := make([]string, 0, len(coords))
	for _, coord := range coords {
		parts = append(parts, formatFloat(coord.Lng)+","+formatFloat(coord.Lat))
	}
	return strings.Join(parts, ";")
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func traceTimestamps(points []routing.TracePoint) (string, bool) {
	parts := make([]string, 0, len(points))
	for _, point := range points {
		if point.Time.IsZero() {
			return "", false
		}
		parts = append(parts, strconv.FormatInt(point.Time.Unix(), 10))
	}
	return strings.Join(parts, ";"), true
}

func traceRadiuses(points []routing.TracePoint) string {
	parts := make([]string, 0, len(points))
	hasRadius := false
	for _, point := range points {
		var radius *float64
		switch {
		case point.Radius != nil:
			radius = point.Radius
		case point.Accuracy != nil:
			radius = point.Accuracy
		}
		if radius == nil {
			parts = append(parts, "unlimited")
			continue
		}
		hasRadius = true
		parts = append(parts, formatFloat(*radius))
	}
	if !hasRadius {
		return ""
	}
	return strings.Join(parts, ";")
}

func indexList(start, count int) string {
	parts := make([]string, 0, count)
	for i := 0; i < count; i++ {
		parts = append(parts, strconv.Itoa(start+i))
	}
	return strings.Join(parts, ";")
}

func warningsFromMessage(message string) []string {
	if message == "" {
		return nil
	}
	return []string{message}
}

func statusError(service, code, message string) error {
	if message == "" {
		return fmt.Errorf("osrm %s: status %s", service, code)
	}
	return fmt.Errorf("osrm %s: status %s: %s", service, code, message)
}

func aggregateRaw(rawResponses [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, raw := range rawResponses {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(raw)
	}
	buf.WriteByte(']')
	return buf.Bytes()
}

func matrixValues(name string, values [][]*float64) ([][]float64, error) {
	matrix := make([][]float64, len(values))
	for i, row := range values {
		matrix[i] = make([]float64, len(row))
		for j, value := range row {
			if value == nil {
				return nil, fmt.Errorf("osrm matrix: null %s value at row %d column %d", name, i, j)
			}
			matrix[i][j] = *value
		}
	}
	return matrix, nil
}

type routeResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Routes  []osrmRoute `json:"routes"`
}

type matchResponse struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Matchings []osrmMatching `json:"matchings"`
}

type nearestResponse struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Waypoints []osrmWaypoint `json:"waypoints"`
}

type tableResponse struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	Durations [][]*float64 `json:"durations"`
	Distances [][]*float64 `json:"distances"`
}

type osrmRoute struct {
	Geometry string  `json:"geometry"`
	Distance float64 `json:"distance"`
	Duration float64 `json:"duration"`
}

type osrmMatching struct {
	Geometry   string   `json:"geometry"`
	Distance   float64  `json:"distance"`
	Duration   float64  `json:"duration"`
	Confidence *float64 `json:"confidence"`
}

type osrmWaypoint struct {
	Location []float64 `json:"location"`
}
