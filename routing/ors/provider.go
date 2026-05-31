package ors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	defaultGeometryFormat = "geojson"
)

// Config contains the minimal settings needed to talk to an OpenRouteService HTTP server.
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
	return routing.ErrNotImplemented
}

func (p *Provider) Capabilities(ctx context.Context) routing.Capabilities {
	return routing.Capabilities{Route: true, MatchTrace: false, Snap: false, Matrix: true}
}

func (p *Provider) Route(ctx context.Context, req routing.RouteRequest) (routing.RouteResult, error) {
	profile := p.profileFor(req.Profile)
	result := routing.RouteResult{Provider: Name, Profile: profile, GeometryFormat: defaultGeometryFormat}
	if req.Start == req.End {
		return result, fmt.Errorf("ors route: start and end coordinates must differ")
	}

	raw, err := p.post(ctx, "directions", profile, "geojson", map[string]any{
		"coordinates": [][]float64{toORSCoordinate(req.Start), toORSCoordinate(req.End)},
	})
	result.RawResponse = raw
	if err != nil {
		return result, err
	}

	var parsed directionsResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("ors route: decode response: %w", err)
	}
	if parsed.Error != nil {
		result.Status = parsed.Error.statusCode()
		result.Warnings = orsWarnings(parsed.Error.Message)
		return result, statusError("route", parsed.Error.statusCode(), parsed.Error.Message)
	}
	if len(parsed.Features) == 0 {
		return result, fmt.Errorf("ors route: no features returned")
	}

	feature := parsed.Features[0]
	geometry, err := compactJSON(feature.Geometry)
	if err != nil {
		return result, fmt.Errorf("ors route: encode geometry: %w", err)
	}
	result.Status = "Ok"
	result.Geometry = geometry
	result.DistanceMeters = feature.Properties.Summary.Distance
	result.DurationSeconds = feature.Properties.Summary.Duration
	result.Warnings = orsWarnings(parsed.Metadata.Query.Warning)
	return result, nil
}

func (p *Provider) MatchTrace(ctx context.Context, req routing.MatchTraceRequest) (routing.MatchTraceResult, error) {
	profile := p.profileFor(req.Profile)
	return routing.MatchTraceResult{Provider: Name, Profile: profile, GeometryFormat: defaultGeometryFormat}, routing.ErrNotImplemented
}

func (p *Provider) Snap(ctx context.Context, req routing.SnapRequest) (routing.SnapResult, error) {
	profile := p.profileFor(req.Profile)
	return routing.SnapResult{Provider: Name, Profile: profile}, routing.ErrNotImplemented
}

func (p *Provider) Matrix(ctx context.Context, req routing.MatrixRequest) (routing.MatrixResult, error) {
	profile := p.profileFor(req.Profile)
	result := routing.MatrixResult{Provider: Name, Profile: profile}
	if len(req.Sources) == 0 {
		return result, fmt.Errorf("ors matrix: at least one source coordinate is required")
	}
	if len(req.Destinations) == 0 {
		return result, fmt.Errorf("ors matrix: at least one destination coordinate is required")
	}

	locations := append([]routing.Coordinate{}, req.Sources...)
	locations = append(locations, req.Destinations...)
	raw, err := p.post(ctx, "matrix", profile, "", map[string]any{
		"locations":    toORSCoordinates(locations),
		"sources":      indexSlice(0, len(req.Sources)),
		"destinations": indexSlice(len(req.Sources), len(req.Destinations)),
		"metrics":      []string{"duration", "distance"},
		"units":        "m",
	})
	result.RawResponse = raw
	if err != nil {
		return result, err
	}

	var parsed matrixResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("ors matrix: decode response: %w", err)
	}
	if parsed.Error != nil {
		result.Status = parsed.Error.statusCode()
		result.Warnings = orsWarnings(parsed.Error.Message)
		return result, statusError("matrix", parsed.Error.statusCode(), parsed.Error.Message)
	}
	durations, err := matrixValues("duration", parsed.Durations)
	if err != nil {
		return result, err
	}
	distances, err := matrixValues("distance", parsed.Distances)
	if err != nil {
		return result, err
	}
	if err := validateMatrixDimensions("duration", durations, len(req.Sources), len(req.Destinations)); err != nil {
		return result, err
	}
	if err := validateMatrixDimensions("distance", distances, len(req.Sources), len(req.Destinations)); err != nil {
		return result, err
	}

	result.Status = "Ok"
	result.DurationMatrix = durations
	result.DistanceMatrix = distances
	result.Warnings = nil
	return result, nil
}

func (p *Provider) post(ctx context.Context, service, profile, suffix string, payload any) ([]byte, error) {
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
	path := strings.TrimRight(endpoint.Path, "/") + "/v2/" + service + "/" + url.PathEscape(profile)
	if suffix != "" {
		path += "/" + suffix
	}
	endpoint.Path = path

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("ors %s: encode request: %w", service, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ors %s: build request: %w", service, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

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
		if providerErr := decodeProviderError(service, responseBody); providerErr != nil {
			return responseBody, providerErr
		}
		return responseBody, fmt.Errorf("ors %s: HTTP %d: %s", service, resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}

func (p *Provider) profileFor(profile string) string {
	if profile != "" {
		return normalizeProfile(profile)
	}
	if p != nil && p.profile != "" {
		return normalizeProfile(p.profile)
	}
	return defaultProfile
}

func normalizeProfile(profile string) string {
	switch profile {
	case "driving", "car":
		return "driving-car"
	case "walking", "foot":
		return "foot-walking"
	case "cycling", "bike":
		return "cycling-regular"
	default:
		return profile
	}
}

func toORSCoordinate(coord routing.Coordinate) []float64 {
	return []float64{coord.Lng, coord.Lat}
}

func toORSCoordinates(coords []routing.Coordinate) [][]float64 {
	locations := make([][]float64, 0, len(coords))
	for _, coord := range coords {
		locations = append(locations, toORSCoordinate(coord))
	}
	return locations
}

func indexSlice(start, count int) []int {
	indexes := make([]int, count)
	for i := range indexes {
		indexes[i] = start + i
	}
	return indexes
}

func compactJSON(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", errors.New("empty geometry")
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func matrixValues(name string, values [][]*float64) ([][]float64, error) {
	matrix := make([][]float64, len(values))
	for i, row := range values {
		matrix[i] = make([]float64, len(row))
		for j, value := range row {
			if value == nil {
				return nil, fmt.Errorf("ors matrix: null %s value at row %d column %d", name, i, j)
			}
			matrix[i][j] = *value
		}
	}
	return matrix, nil
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

func decodeProviderError(service string, raw []byte) error {
	var parsed struct {
		Error *orsError `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Error == nil {
		return nil
	}
	return statusError(service, parsed.Error.statusCode(), parsed.Error.Message)
}

func statusError(service, code, message string) error {
	if message == "" {
		return fmt.Errorf("ors %s: status %s", service, code)
	}
	return fmt.Errorf("ors %s: status %s: %s", service, code, message)
}

func orsWarnings(messages ...string) []string {
	warnings := make([]string, 0, len(messages))
	for _, message := range messages {
		if message != "" {
			warnings = append(warnings, message)
		}
	}
	return warnings
}

type directionsResponse struct {
	Features []orsFeature `json:"features"`
	Metadata orsMetadata  `json:"metadata"`
	Error    *orsError    `json:"error"`
}

type orsFeature struct {
	Geometry   json.RawMessage      `json:"geometry"`
	Properties orsFeatureProperties `json:"properties"`
}

type orsFeatureProperties struct {
	Summary orsSummary `json:"summary"`
}

type orsSummary struct {
	Distance float64 `json:"distance"`
	Duration float64 `json:"duration"`
}

type matrixResponse struct {
	Durations [][]*float64 `json:"durations"`
	Distances [][]*float64 `json:"distances"`
	Metadata  orsMetadata  `json:"metadata"`
	Error     *orsError    `json:"error"`
}

type orsMetadata struct {
	Service     string   `json:"service"`
	Attribution string   `json:"attribution"`
	Query       orsQuery `json:"query"`
}

type orsQuery struct {
	Warning string `json:"warning"`
}

type orsError struct {
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
}

func (err orsError) statusCode() string {
	if len(err.Code) == 0 {
		return ""
	}
	var codeString string
	if unmarshalErr := json.Unmarshal(err.Code, &codeString); unmarshalErr == nil {
		return codeString
	}
	return strings.Trim(string(err.Code), `"`)
}
