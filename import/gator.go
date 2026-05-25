package importers

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/MickMake/GoTravel/storage"
)

const gatorTimeLayout = "2006-01-02 15:04:05"

type Gator struct{}

func (Gator) Import(r io.Reader, source string) storage.ImportResult {
	result := storage.ImportResult{Format: "gator", SourceFile: source}
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		result.Errors = append(result.Errors, storage.ImportError{SourceFile: source, SourceLine: 1, Format: "gator", Error: err.Error()})
		return result
	}

	col := map[string]int{}
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, required := range []string{"dt", "lat", "lng", "altitude", "angle", "speed", "params"} {
		if _, ok := col[required]; !ok {
			result.Errors = append(result.Errors, storage.ImportError{SourceFile: source, SourceLine: 1, Format: "gator", RawRow: strings.Join(header, ","), Error: fmt.Sprintf("missing required column %q", required)})
			return result
		}
	}

	line := 1
	for {
		row, err := reader.Read()
		line++
		if err == io.EOF {
			break
		}
		if err != nil {
			result.RowsSeen++
			result.Errors = append(result.Errors, storage.ImportError{SourceFile: source, SourceLine: line, Format: "gator", Error: err.Error()})
			continue
		}
		result.RowsSeen++

		p, err := parseGatorRow(row, col, source, line)
		if err != nil {
			result.Errors = append(result.Errors, storage.ImportError{SourceFile: source, SourceLine: line, Format: "gator", RawRow: strings.Join(row, ","), Error: err.Error()})
			continue
		}
		result.Points = append(result.Points, p)
	}
	return result
}

func parseGatorRow(row []string, col map[string]int, source string, line int) (storage.Point, error) {
	get := func(name string) (string, error) {
		idx := col[name]
		if idx >= len(row) {
			return "", fmt.Errorf("column %q missing from row", name)
		}
		return strings.TrimSpace(row[idx]), nil
	}

	dtRaw, _ := get("dt")
	dt, err := time.ParseInLocation(gatorTimeLayout, dtRaw, time.Local)
	if err != nil {
		return storage.Point{}, fmt.Errorf("invalid dt %q", dtRaw)
	}

	lat, err := parseFloatField(get, "lat")
	if err != nil {
		return storage.Point{}, err
	}
	lng, err := parseFloatField(get, "lng")
	if err != nil {
		return storage.Point{}, err
	}
	altitude, err := parseFloatField(get, "altitude")
	if err != nil {
		return storage.Point{}, err
	}
	angle, err := parseFloatField(get, "angle")
	if err != nil {
		return storage.Point{}, err
	}
	speed, err := parseFloatField(get, "speed")
	if err != nil {
		return storage.Point{}, err
	}
	params, _ := get("params")

	return storage.Point{
		DT:         dt,
		Lat:        lat,
		Lng:        lng,
		Altitude:   altitude,
		Angle:      angle,
		Speed:      speed,
		Params:     params,
		Format:     "gator",
		SourceFile: source,
		SourceLine: line,
	}, nil
}

func parseFloatField(get func(string) (string, error), name string) (float64, error) {
	raw, err := get(name)
	if err != nil {
		return 0, err
	}
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", name, raw)
	}
	return value, nil
}
