package routing

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestRouteGeometryAsGeoJSONPassThrough(t *testing.T) {
	geometry, err := RouteGeometryAsGeoJSON("application/geo+json", `{"type":"LineString","coordinates":[[151,-33],[151.1,-33.1]]}`)
	if err != nil {
		t.Fatalf("RouteGeometryAsGeoJSON returned error: %v", err)
	}
	object := geometry.(map[string]any)
	if object["type"] != "LineString" {
		t.Fatalf("type = %v, want LineString", object["type"])
	}
}

func TestDecodeEncodedPolylinePrecision5(t *testing.T) {
	coordinates, err := DecodeEncodedPolyline("_p~iF~ps|U_ulLnnqC_mqNvxq`@", 5)
	if err != nil {
		t.Fatalf("DecodeEncodedPolyline returned error: %v", err)
	}
	assertCoordinatesNear(t, coordinates, []Coordinate{
		{Lng: -120.2, Lat: 38.5},
		{Lng: -120.95, Lat: 40.7},
		{Lng: -126.453, Lat: 43.252},
	})
}

func TestRouteGeometryAsGeoJSONPolyline5(t *testing.T) {
	geometry, err := RouteGeometryAsGeoJSON("encoded-polyline", "_p~iF~ps|U_ulLnnqC_mqNvxq`@")
	if err != nil {
		t.Fatalf("RouteGeometryAsGeoJSON returned error: %v", err)
	}
	object := geometry.(map[string]any)
	if object["type"] != "LineString" {
		t.Fatalf("type = %v, want LineString", object["type"])
	}
	coordinates := object["coordinates"].([][]float64)
	if got, want := len(coordinates), 3; got != want {
		t.Fatalf("coordinate count = %d, want %d", got, want)
	}
	if coordinates[0][0] != -120.2 || coordinates[0][1] != 38.5 {
		t.Fatalf("first coordinate = %v", coordinates[0])
	}
}

func TestRouteGeometryAsGeoJSONPolyline6(t *testing.T) {
	encoded := encodePolylineForTest([]Coordinate{
		{Lng: 151.2093, Lat: -33.8688},
		{Lng: 151.2152, Lat: -33.8568},
	}, 6)
	geometry, err := RouteGeometryAsGeoJSON("polyline6", encoded)
	if err != nil {
		t.Fatalf("RouteGeometryAsGeoJSON returned error: %v", err)
	}
	object := geometry.(map[string]any)
	if object["type"] != "LineString" {
		t.Fatalf("type = %v, want LineString", object["type"])
	}
	coordinates := object["coordinates"].([][]float64)
	assertCoordinatesNear(t, []Coordinate{
		{Lng: coordinates[0][0], Lat: coordinates[0][1]},
		{Lng: coordinates[1][0], Lat: coordinates[1][1]},
	}, []Coordinate{
		{Lng: 151.2093, Lat: -33.8688},
		{Lng: 151.2152, Lat: -33.8568},
	})
}

func TestRouteGeometryCoordinatesFromGeoJSONFeature(t *testing.T) {
	coordinates, err := RouteGeometryCoordinates("geojson", `{"type":"Feature","geometry":{"type":"LineString","coordinates":[[151,-33],[151.1,-33.1]]},"properties":{}}`)
	if err != nil {
		t.Fatalf("RouteGeometryCoordinates returned error: %v", err)
	}
	want := []Coordinate{{Lng: 151, Lat: -33}, {Lng: 151.1, Lat: -33.1}}
	if !reflect.DeepEqual(coordinates, want) {
		t.Fatalf("coordinates = %#v, want %#v", coordinates, want)
	}
}

func TestRouteGeometryAsGeoJSONUnsupportedFormat(t *testing.T) {
	_, err := RouteGeometryAsGeoJSON("wibble", "abc")
	if err == nil || !strings.Contains(err.Error(), "unsupported route geometry format") {
		t.Fatalf("error = %v, want unsupported format", err)
	}
}

func TestDecodeEncodedPolylineMalformed(t *testing.T) {
	_, err := DecodeEncodedPolyline("_", 5)
	if err == nil || !strings.Contains(err.Error(), "malformed encoded polyline") {
		t.Fatalf("error = %v, want malformed encoded polyline", err)
	}
}

func assertCoordinatesNear(t *testing.T, got, want []Coordinate) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("coordinate count = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if math.Abs(got[i].Lng-want[i].Lng) > 0.000001 || math.Abs(got[i].Lat-want[i].Lat) > 0.000001 {
			t.Fatalf("coordinate %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func encodePolylineForTest(coordinates []Coordinate, precision int) string {
	factor := math.Pow10(precision)
	lastLat := 0
	lastLng := 0
	var b strings.Builder
	for _, coordinate := range coordinates {
		lat := int(math.Round(coordinate.Lat * factor))
		lng := int(math.Round(coordinate.Lng * factor))
		encodePolylineValueForTest(&b, lat-lastLat)
		encodePolylineValueForTest(&b, lng-lastLng)
		lastLat = lat
		lastLng = lng
	}
	return b.String()
}

func encodePolylineValueForTest(b *strings.Builder, value int) {
	value <<= 1
	if value < 0 {
		value = ^value
	}
	for value >= 0x20 {
		b.WriteByte(byte((0x20 | (value & 0x1f)) + 63))
		value >>= 5
	}
	b.WriteByte(byte(value + 63))
}
