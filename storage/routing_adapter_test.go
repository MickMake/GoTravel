package storage

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestMatchTraceRequestFromPoints(t *testing.T) {
	radius := 12.5
	points := []Point{
		{DT: time.Unix(1700000000, 0), Lat: -33.8, Lng: 151.2},
		{DT: time.Unix(1700000060, 0), Lat: -33.9, Lng: 151.3},
	}

	request, err := MatchTraceRequestFromPoints(points, MatchTraceOptions{Profile: "driving", Radius: &radius})
	if err != nil {
		t.Fatalf("MatchTraceRequestFromPoints() err=%v", err)
	}
	if request.Profile != "driving" {
		t.Fatalf("Profile=%q", request.Profile)
	}
	if len(request.Points) != 2 {
		t.Fatalf("points=%+v", request.Points)
	}
	if request.Points[0].Lat != -33.8 || request.Points[0].Lng != 151.2 || !request.Points[0].Time.Equal(points[0].DT) {
		t.Fatalf("point 0=%+v", request.Points[0])
	}
	if request.Points[0].Radius == nil || *request.Points[0].Radius != radius {
		t.Fatalf("radius=%v", request.Points[0].Radius)
	}
	if request.Points[1].Radius == nil || *request.Points[1].Radius != radius {
		t.Fatalf("radius=%v", request.Points[1].Radius)
	}
}

func TestMatchTraceRequestFromPointsWithoutRadius(t *testing.T) {
	points := []Point{
		{DT: time.Unix(1700000000, 0), Lat: -33.8, Lng: 151.2},
		{DT: time.Unix(1700000060, 0), Lat: -33.9, Lng: 151.3},
	}

	request, err := MatchTraceRequestFromPoints(points, MatchTraceOptions{})
	if err != nil {
		t.Fatalf("MatchTraceRequestFromPoints() err=%v", err)
	}
	if request.Points[0].Radius != nil || request.Points[1].Radius != nil {
		t.Fatalf("unexpected radiuses: %+v", request.Points)
	}
}

func TestMatchTraceRequestFromPointsRejectsTooFewPoints(t *testing.T) {
	_, err := MatchTraceRequestFromPoints([]Point{{DT: time.Unix(1700000000, 0), Lat: -33.8, Lng: 151.2}}, MatchTraceOptions{})
	if err == nil || !strings.Contains(err.Error(), "at least two points") {
		t.Fatalf("err=%v", err)
	}
}

func TestMatchTraceRequestFromPointsRejectsZeroTimestamp(t *testing.T) {
	points := []Point{
		{DT: time.Unix(1700000000, 0), Lat: -33.8, Lng: 151.2},
		{Lat: -33.9, Lng: 151.3},
	}

	_, err := MatchTraceRequestFromPoints(points, MatchTraceOptions{})
	if err == nil || !strings.Contains(err.Error(), "zero timestamp") {
		t.Fatalf("err=%v", err)
	}
}

func TestMatchTraceRequestFromPointsRejectsInvalidCoordinates(t *testing.T) {
	base := []Point{
		{DT: time.Unix(1700000000, 0), Lat: -33.8, Lng: 151.2},
		{DT: time.Unix(1700000060, 0), Lat: -33.9, Lng: 151.3},
	}

	cases := []struct {
		name    string
		lat     float64
		lng     float64
		message string
	}{
		{name: "latitude too high", lat: 91, lng: 151.3, message: "latitude"},
		{name: "longitude too high", lat: -33.9, lng: 181, message: "longitude"},
		{name: "latitude NaN", lat: math.NaN(), lng: 151.3, message: "latitude"},
		{name: "longitude NaN", lat: -33.9, lng: math.NaN(), message: "longitude"},
		{name: "latitude infinity", lat: math.Inf(1), lng: 151.3, message: "latitude"},
		{name: "longitude infinity", lat: -33.9, lng: math.Inf(-1), message: "longitude"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			points := append([]Point(nil), base...)
			points[1].Lat = tc.lat
			points[1].Lng = tc.lng
			_, err := MatchTraceRequestFromPoints(points, MatchTraceOptions{})
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("err=%v, want %q", err, tc.message)
			}
		})
	}
}
