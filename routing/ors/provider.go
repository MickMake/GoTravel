package ors

import (
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
	Name                  = "ors"
	defaultBaseURL        = "http://127.0.0.1:8080/ors"
	defaultProfile        = "driving-car"
	defaultGeometryFormat = "polyline5"
)

// Config contains the minimal settings needed to talk to an ORS HTTP server.
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

func New() *Provider { return NewWithConfig(Config{}) }

func NewWithConfig(cfg Config) *Provider {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	profile := orsProfile(cfg.Profile)
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
	_, err := p.get(ctx, "health")
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
		return result, fmt.Errorf("ors route: start and end coordinates must differ")
	}
	raw, err := p.post(ctx, "directions", profile, routeRequestBody{Coordinates: coordinates([]routing.Coordinate{req.Start, req.End}), GeometryFormat: defaultGeometryFormat})
	result.RawResponse = raw
	if err != nil {
		return result, err
	}
	var parsed routeResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("ors route: decode response: %w", err)
	}
	if err := applyErrorStatus("route", &result.Status, &result.Warnings, parsed.Error); err != nil {
		return result, err
	}
	if len(parsed.Routes) == 0 {
		return result, fmt.Errorf("ors route: no routes returned")
	}
	route := parsed.Routes[0]
	result.Status = "Ok"
	result.Geometry = route.Geometry
	result.DistanceMeters = route.Summary.Distance
	result.DurationSeconds = route.Summary.Duration
	result.Warnings = append(result.Warnings, warnings(parsed.Info.Warnings)...)
	return result, nil
}

func (p *Provider) MatchTrace(ctx context.Context, req routing.MatchTraceRequest) (routing.MatchTraceResult, error) {
	profile := p.profileFor(req.Profile)
	result := routing.MatchTraceResult{Provider: Name, Profile: profile, GeometryFormat: defaultGeometryFormat}
	if len(req.Points) < 2 {
		return result, fmt.Errorf("ors match: at least two trace points are required")
	}
	coords := make([]routing.Coordinate, 0, len(req.Points))
	for _, point := range req.Points {
		coords = append(coords, point.Coordinate)
	}
	raw, err := p.post(ctx, "directions", profile, routeRequestBody{Coordinates: coordinates(coords), GeometryFormat: defaultGeometryFormat})
	result.RawResponse = raw
	if err != nil {
		return result, err
	}
	var parsed routeResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("ors match: decode response: %w", err)
	}
	if err := applyErrorStatus("match", &result.Status, &result.Warnings, parsed.Error); err != nil {
		return result, err
	}
	if len(parsed.Routes) == 0 {
		return result, fmt.Errorf("ors match: no routes returned")
	}
	route := parsed.Routes[0]
	result.Status = "Ok"
	result.Geometry = route.Geometry
	result.DistanceMeters = route.Summary.Distance
	result.DurationSeconds = route.Summary.Duration
	result.Warnings = append(result.Warnings, warnings(parsed.Info.Warnings)...)
	return result, nil
}

func (p *Provider) Snap(ctx context.Context, req routing.SnapRequest) (routing.SnapResult, error) {
	profile := p.profileFor(req.Profile)
	result := routing.SnapResult{Provider: Name, Profile: profile, Status: "Ok"}
	if len(req.Coordinates) == 0 {
		return result, fmt.Errorf("ors snap: at least one coordinate is required")
	}
	raw, err := p.post(ctx, "snap", profile, snapRequestBody{Locations: coordinates(req.Coordinates)})
	result.RawResponse = raw
	if err != nil {
		return result, err
	}
	var parsed snapResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("ors snap: decode response: %w", err)
	}
	if err := applyErrorStatus("snap", &result.Status, &result.Warnings, parsed.Error); err != nil {
		return result, err
	}
	if len(parsed.Locations) != len(req.Coordinates) {
		return result, fmt.Errorf("ors snap: returned %d locations, want %d", len(parsed.Locations), len(req.Coordinates))
	}
	for i, location := range parsed.Locations {
		if len(location.Location) < 2 {
			return result, fmt.Errorf("ors snap: no snapped coordinate returned for location %d", i)
		}
		result.Points = append(result.Points, routing.Coordinate{Lng: location.Location[0], Lat: location.Location[1]})
	}
	result.Warnings = append(result.Warnings, warnings(parsed.Info.Warnings)...)
	return result, nil
}

func (p *Provider) Matrix(ctx context.Context, req routing.MatrixRequest) (routing.MatrixResult, error) {
	profile := p.profileFor(req.Profile)
	result := routing.MatrixResult{Provider: Name, Profile: profile, Status: "Ok"}
	if len(req.Sources) == 0 {
		return result, fmt.Errorf("ors matrix: at least one source coordinate is required")
	}
	if len(req.Destinations) == 0 {
		return result, fmt.Errorf("ors matrix: at least one destination coordinate is required")
	}
	locations := append([]routing.Coordinate{}, req.Sources...)
	locations = append(locations, req.Destinations...)
	raw, err := p.post(ctx, "matrix", profile, matrixRequestBody{Locations: coordinates(locations), Sources: indexes(0, len(req.Sources)), Destinations: indexes(len(req.Sources), len(req.Destinations)), Metrics: []string{"duration", "distance"}, Units: "m"})
	result.RawResponse = raw
	if err != nil {
		return result, err
	}
	var parsed matrixResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("ors matrix: decode response: %w", err)
	}
	if err := applyErrorStatus("matrix", &result.Status, &result.Warnings, parsed.Error); err != nil {
		return result, err
	}
	if err := validateMatrixDimensions("duration", parsed.Durations, len(req.Sources), len(req.Destinations)); err != nil {
		return result, err
	}
	if err := validateMatrixDimensions("distance", parsed.Distances, len(req.Sources), len(req.Destinations)); err != nil {
		return result, err
	}
	result.DurationMatrix = parsed.Durations
	result.DistanceMatrix = parsed.Distances
	result.Warnings = append(result.Warnings, warnings(parsed.Info.Warnings)...)
	return result, nil
}

func (p *Provider) get(ctx context.Context, service string) ([]byte, error) { return p.do(ctx, http.MethodGet, service, "", nil) }
func (p *Provider) post(ctx context.Context, service, profile string, payload any) ([]byte, error) {
	return p.do(ctx, http.MethodPost, service, profile, payload)
}

func (p *Provider) do(ctx context.Context, method, service, profile string, payload any) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("ors %s: nil provider", service)
	}
	if p.baseURL == "" {
		return nil, fmt.Errorf("ors %s: base URL is required", service)
	}
	endpoint, err := url.Parse(p.baseURL)
	if err != nil {
		return nil, fmt.Errorf("ors %s: invalid base URL: %w", service, err)
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/v2/" + service
	if profile != "" {
		endpoint.Path += "/" + url.PathEscape(profile)
		if method == http.MethodPost {
			endpoint.Path += "/json"
		}
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("ors %s: encode request: %w", service, err)
		}
		body = strings.NewReader(string(encoded))
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("ors %s: build request: %w", service, err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ors %s: request failed: %w", service, err)
	}
	defer resp.Body.Close()
	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return responseBody, fmt.Errorf("ors %s: read response: %w", service, readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseBody, fmt.Errorf("ors %s: HTTP %d: %s", service, resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}

func (p *Provider) profileFor(profile string) string {
	if profile != "" {
		return orsProfile(profile)
	}
	if p != nil && p.profile != "" {
		return p.profile
	}
	return defaultProfile
}

func orsProfile(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "":
		return ""
	case "driving":
		return "driving-car"
	case "walking":
		return "foot-walking"
	case "cycling", "bicycle", "bike":
		return "cycling-regular"
	default:
		return strings.TrimSpace(profile)
	}
}

func coordinates(coords []routing.Coordinate) [][]float64 {
	values := make([][]float64, 0, len(coords))
	for _, coord := range coords {
		values = append(values, []float64{coord.Lng, coord.Lat})
	}
	return values
}

func indexes(start, count int) []int {
	values := make([]int, 0, count)
	for i := 0; i < count; i++ {
		values = append(values, start+i)
	}
	return values
}

func applyErrorStatus(service string, status *string, targetWarnings *[]string, providerError *providerError) error {
	if providerError == nil {
		return nil
	}
	*status = fmt.Sprintf("%d", providerError.Code)
	if providerError.Message != "" {
		*targetWarnings = append(*targetWarnings, providerError.Message)
	}
	return statusError(service, providerError.Code, providerError.Message)
}

func warnings(values []providerWarning) []string {
	messages := make([]string, 0, len(values))
	for _, value := range values {
		if value.Message != "" {
			messages = append(messages, value.Message)
		}
	}
	return messages
}

func statusError(service string, code int, message string) error {
	if message == "" {
		return fmt.Errorf("ors %s: status %d", service, code)
	}
	return fmt.Errorf("ors %s: status %d: %s", service, code, message)
}

func validateMatrixDimensions(name string, matrix [][]float64, sources, destinations int) error {
	if len(matrix) != sources {
		return fmt.Errorf("ors matrix: %s matrix has %d rows, want %d", name, len(matrix), sources)
	}
	for i, row := range matrix {
		if len(row) != destinations {
			return fmt.Errorf("ors matrix: %s matrix row %d has %d columns, want %d", name, i, len(row), destinations)
		}
	}
	return nil
}

type routeRequestBody struct {
	Coordinates    [][]float64 `json:"coordinates"`
	GeometryFormat string      `json:"geometry_format"`
}
type snapRequestBody struct{ Locations [][]float64 `json:"locations"` }
type matrixRequestBody struct {
	Locations    [][]float64 `json:"locations"`
	Sources      []int       `json:"sources"`
	Destinations []int       `json:"destinations"`
	Metrics      []string    `json:"metrics"`
	Units        string      `json:"units"`
}
type routeResponse struct {
	Routes []orsRoute     `json:"routes"`
	Error  *providerError `json:"error"`
	Info   responseInfo   `json:"info"`
}
type snapResponse struct {
	Locations []snapLocation `json:"locations"`
	Error     *providerError `json:"error"`
	Info      responseInfo   `json:"info"`
}
type matrixResponse struct {
	Durations [][]float64    `json:"durations"`
	Distances [][]float64    `json:"distances"`
	Error     *providerError `json:"error"`
	Info      responseInfo   `json:"info"`
}
type orsRoute struct {
	Geometry string     `json:"geometry"`
	Summary  orsSummary `json:"summary"`
}
type orsSummary struct {
	Distance float64 `json:"distance"`
	Duration float64 `json:"duration"`
}
type snapLocation struct{ Location []float64 `json:"location"` }
type providerError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type responseInfo struct{ Warnings []providerWarning `json:"warnings"` }
type providerWarning struct{ Message string `json:"message"` }
