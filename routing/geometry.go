package routing

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode"
)

// Coordinate is a provider-neutral longitude/latitude pair.
type Coordinate struct {
	Lon float64
	Lat float64
}

// RouteGeometryAsGeoJSON converts a stored route geometry into a GeoJSON geometry object.
func RouteGeometryAsGeoJSON(format, geometry string) (any, error) {
	geometry = strings.TrimSpace(geometry)
	if geometry == "" {
		return nil, fmt.Errorf("route geometry is empty")
	}

	if routeGeometryIsGeoJSON(format, geometry) {
		var decoded any
		if err := json.Unmarshal([]byte(geometry), &decoded); err != nil {
			return nil, fmt.Errorf("route geometry is not valid GeoJSON: %w", err)
		}
		return decoded, nil
	}

	precision, ok := encodedPolylinePrecision(format)
	if !ok {
		return nil, fmt.Errorf("unsupported route geometry format %q", format)
	}
	coordinates, err := DecodeEncodedPolyline(geometry, precision)
	if err != nil {
		return nil, err
	}
	return lineStringGeometry(coordinates), nil
}

// RouteGeometryCoordinates converts a stored route geometry into longitude/latitude coordinates.
func RouteGeometryCoordinates(format, geometry string) ([]Coordinate, error) {
	geometry = strings.TrimSpace(geometry)
	if geometry == "" {
		return nil, fmt.Errorf("route geometry is empty")
	}

	if routeGeometryIsGeoJSON(format, geometry) {
		geojson, err := RouteGeometryAsGeoJSON(format, geometry)
		if err != nil {
			return nil, err
		}
		return coordinatesFromGeoJSON(geojson)
	}

	precision, ok := encodedPolylinePrecision(format)
	if !ok {
		return nil, fmt.Errorf("unsupported route geometry format %q", format)
	}
	return DecodeEncodedPolyline(geometry, precision)
}

// DecodeEncodedPolyline decodes a Google encoded polyline with the supplied precision.
func DecodeEncodedPolyline(encoded string, precision int) ([]Coordinate, error) {
	if precision <= 0 {
		return nil, fmt.Errorf("polyline precision must be positive")
	}
	if encoded == "" {
		return nil, fmt.Errorf("encoded polyline is empty")
	}

	factor := math.Pow10(precision)
	index := 0
	lat := 0
	lon := 0
	coordinates := make([]Coordinate, 0)
	for index < len(encoded) {
		deltaLat, next, err := decodePolylineValue(encoded, index)
		if err != nil {
			return nil, err
		}
		index = next
		if index >= len(encoded) {
			return nil, fmt.Errorf("malformed encoded polyline: missing longitude")
		}
		deltaLon, next, err := decodePolylineValue(encoded, index)
		if err != nil {
			return nil, err
		}
		index = next

		lat += deltaLat
		lon += deltaLon
		coordinates = append(coordinates, Coordinate{
			Lon: float64(lon) / factor,
			Lat: float64(lat) / factor,
		})
	}
	if len(coordinates) == 0 {
		return nil, fmt.Errorf("encoded polyline contains no coordinates")
	}
	return coordinates, nil
}

func decodePolylineValue(encoded string, index int) (int, int, error) {
	result := 0
	shift := 0
	for index < len(encoded) {
		b := int(encoded[index]) - 63
		if b < 0 {
			return 0, index, fmt.Errorf("malformed encoded polyline: invalid character")
		}
		index++
		result |= (b & 0x1f) << shift
		shift += 5
		if b < 0x20 {
			if result&1 != 0 {
				return ^(result >> 1), index, nil
			}
			return result >> 1, index, nil
		}
		if shift > 30 {
			return 0, index, fmt.Errorf("malformed encoded polyline: value is too large")
		}
	}
	return 0, index, fmt.Errorf("malformed encoded polyline: unterminated value")
}

func routeGeometryIsGeoJSON(format, geometry string) bool {
	normalized := normalizeGeometryFormat(format)
	return strings.Contains(normalized, "geojson") || strings.HasPrefix(strings.TrimSpace(geometry), "{")
}

func encodedPolylinePrecision(format string) (int, bool) {
	switch normalizeGeometryFormat(format) {
	case "polyline", "encodedpolyline", "encodedpolyline5", "polyline5":
		return 5, true
	case "polyline6", "encodedpolyline6":
		return 6, true
	default:
		return 0, false
	}
}

func normalizeGeometryFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	var b strings.Builder
	for _, r := range format {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func lineStringGeometry(coordinates []Coordinate) map[string]any {
	return map[string]any{
		"type":        "LineString",
		"coordinates": coordinatePairs(coordinates),
	}
}

func coordinatePairs(coordinates []Coordinate) [][]float64 {
	pairs := make([][]float64, 0, len(coordinates))
	for _, coordinate := range coordinates {
		pairs = append(pairs, []float64{coordinate.Lon, coordinate.Lat})
	}
	return pairs
}

func coordinatesFromGeoJSON(value any) ([]Coordinate, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("GeoJSON geometry must be an object")
	}
	geometryType, _ := object["type"].(string)
	if strings.EqualFold(geometryType, "Feature") {
		return coordinatesFromGeoJSON(object["geometry"])
	}
	if !strings.EqualFold(geometryType, "LineString") {
		return nil, fmt.Errorf("GeoJSON geometry type %q is not supported for route coordinates", geometryType)
	}
	rawCoordinates, ok := object["coordinates"].([]any)
	if !ok {
		return nil, fmt.Errorf("GeoJSON LineString coordinates must be an array")
	}
	coordinates := make([]Coordinate, 0, len(rawCoordinates))
	for _, rawPair := range rawCoordinates {
		pair, ok := rawPair.([]any)
		if !ok || len(pair) < 2 {
			return nil, fmt.Errorf("GeoJSON LineString coordinate must contain longitude and latitude")
		}
		lon, ok := pair[0].(float64)
		if !ok {
			return nil, fmt.Errorf("GeoJSON longitude must be numeric")
		}
		lat, ok := pair[1].(float64)
		if !ok {
			return nil, fmt.Errorf("GeoJSON latitude must be numeric")
		}
		coordinates = append(coordinates, Coordinate{Lon: lon, Lat: lat})
	}
	if len(coordinates) == 0 {
		return nil, fmt.Errorf("GeoJSON LineString contains no coordinates")
	}
	return coordinates, nil
}
