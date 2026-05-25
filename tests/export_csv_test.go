package tests

import (
	"bytes"
	"strings"
	"testing"
	"time"

	exporters "github.com/MickMake/GoTravel/export"
	"github.com/MickMake/GoTravel/storage"
)

func TestCSVExport(t *testing.T) {
	exp, err := exporters.New("csv")
	if err != nil {
		t.Fatal(err)
	}
	dt, _ := time.Parse("2006-01-02 15:04:05", "2025-05-01 10:11:05")
	points := []storage.Point{{DT: dt, Lat: -33.74158, Lng: 151.047845, Altitude: 189, Angle: 73, Speed: 0, Params: "gpslev=17"}}
	var buf bytes.Buffer
	if err := exp.Export(&buf, points); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "2025-05-01 10:11:05") {
		t.Fatalf("missing exported timestamp: %s", buf.String())
	}
}
