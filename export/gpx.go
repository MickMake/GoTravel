package exporters

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/MickMake/GoTravel/storage"
)

type GPX struct{}

type gpxDocument struct {
	XMLName xml.Name `xml:"gpx"`
	Version string   `xml:"version,attr"`
	Creator string   `xml:"creator,attr"`
	XMLNS   string   `xml:"xmlns,attr"`
	Track   gpxTrack `xml:"trk"`
}

type gpxTrack struct {
	Name    string     `xml:"name,omitempty"`
	Segment gpxSegment `xml:"trkseg"`
}

type gpxSegment struct {
	Points []gpxPoint `xml:"trkpt"`
}

type gpxPoint struct {
	Lat  string `xml:"lat,attr"`
	Lon  string `xml:"lon,attr"`
	Ele  string `xml:"ele,omitempty"`
	Time string `xml:"time"`
}

func (GPX) Export(w io.Writer, points []storage.Point) error {
	if len(points) == 0 {
		return fmt.Errorf("no points to export")
	}

	doc := gpxDocument{
		Version: "1.1",
		Creator: "GoTravel",
		XMLNS:   "http://www.topografix.com/GPX/1/1",
		Track: gpxTrack{
			Name:    "GoTravel export",
			Segment: gpxSegment{Points: make([]gpxPoint, 0, len(points))},
		},
	}

	for _, p := range points {
		doc.Track.Segment.Points = append(doc.Track.Segment.Points, gpxPoint{
			Lat:  fmt.Sprintf("%.7f", p.Lat),
			Lon:  fmt.Sprintf("%.7f", p.Lng),
			Ele:  fmt.Sprintf("%.0f", p.Altitude),
			Time: p.DT.UTC().Format(time.RFC3339),
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
	_, err := io.WriteString(w, "\n")
	return err
}
