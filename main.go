package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const defaultDB = "gotravel.sqlite"
const timeLayout = "2006-01-02 15:04:05"

type Point struct {
	Source   string
	Format   string
	DT       string
	Lat      string
	Lng      string
	Altitude string
	Angle    string
	Speed    string
	Params   string
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "import":
		err = cmdImport(os.Args[2:])
	case "export":
		err = cmdExport(os.Args[2:])
	case "help", "--help", "-h":
		usage()
		return
	default:
		err = fmt.Errorf("unknown command: %s", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `GoTravel 0.1

Usage:
  GoTravel import [--db gotravel.sqlite] <gator|google> <input.csv> [...]
  GoTravel import [--db gotravel.sqlite] <gator|google> -
  GoTravel export [--db gotravel.sqlite] <output.csv|-> [--start value] [--stop value]

Examples:
  GoTravel import gator trip1.csv trip2.csv
  cat trip.csv | GoTravel import gator -
  GoTravel export - --start 2025-05 --stop 2025-06

Notes:
  - Google import is reserved but not implemented yet.
  - Export currently writes the staged SQLite columns as CSV.
  - Date filters accept YYYY, YYYY-MM, YYYY-MM-DD, or YYYY-MM-DD HH:MM:SS.
`)
}

func cmdImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", defaultDB, "SQLite database path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) < 2 {
		return errors.New("import needs a format and at least one input file")
	}

	format := strings.ToLower(rest[0])
	if format != "gator" && format != "google" {
		return fmt.Errorf("unsupported import format: %s", format)
	}
	if format == "google" {
		return errors.New("google import is not implemented yet")
	}

	db, err := openDB(*dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	var total ImportStats
	for _, name := range rest[1:] {
		stats, err := importOne(db, format, name)
		if err != nil {
			return err
		}
		total.Read += stats.Read
		total.Inserted += stats.Inserted
		total.Duplicates += stats.Duplicates
	}

	fmt.Fprintf(os.Stderr, "imported: read=%d inserted=%d duplicates=%d db=%s\n", total.Read, total.Inserted, total.Duplicates, *dbPath)
	return nil
}

type ImportStats struct {
	Read       int
	Inserted   int
	Duplicates int
}

func importOne(db *sql.DB, format, name string) (ImportStats, error) {
	var r io.Reader
	var closeFn func() error

	if name == "-" {
		r = os.Stdin
		name = "stdin"
		closeFn = func() error { return nil }
	} else {
		f, err := os.Open(name)
		if err != nil {
			return ImportStats{}, err
		}
		r = f
		closeFn = f.Close
	}
	defer closeFn()

	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return ImportStats{}, err
	}
	col := columns(header)

	required := []string{"dt", "lat", "lng", "altitude", "angle", "speed", "params"}
	for _, h := range required {
		if _, ok := col[h]; !ok {
			return ImportStats{}, fmt.Errorf("%s: missing required column %q", name, h)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return ImportStats{}, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
INSERT OR IGNORE INTO points
(source, format, dt, lat, lng, altitude, angle, speed, params, point_hash)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return ImportStats{}, err
	}
	defer stmt.Close()

	stats := ImportStats{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stats, err
		}
		stats.Read++

		p := Point{
			Source:   sourceName(name),
			Format:   format,
			DT:       cell(row, col["dt"]),
			Lat:      cell(row, col["lat"]),
			Lng:      cell(row, col["lng"]),
			Altitude: cell(row, col["altitude"]),
			Angle:    cell(row, col["angle"]),
			Speed:    cell(row, col["speed"]),
			Params:   cell(row, col["params"]),
		}

		if _, err := time.Parse(timeLayout, p.DT); err != nil {
			return stats, fmt.Errorf("%s row %d: invalid dt %q", name, stats.Read+1, p.DT)
		}
		if _, err := strconv.ParseFloat(p.Lat, 64); err != nil {
			return stats, fmt.Errorf("%s row %d: invalid lat %q", name, stats.Read+1, p.Lat)
		}
		if _, err := strconv.ParseFloat(p.Lng, 64); err != nil {
			return stats, fmt.Errorf("%s row %d: invalid lng %q", name, stats.Read+1, p.Lng)
		}

		res, err := stmt.Exec(p.Source, p.Format, p.DT, p.Lat, p.Lng, nullable(p.Altitude), nullable(p.Angle), nullable(p.Speed), p.Params, p.Hash())
		if err != nil {
			return stats, err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			stats.Duplicates++
		} else {
			stats.Inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return stats, err
	}
	return stats, nil
}

func cmdExport(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dbPath := fs.String("db", defaultDB, "SQLite database path")
	start := fs.String("start", "", "inclusive start date/time")
	stop := fs.String("stop", "", "inclusive stop date/time")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) != 1 {
		return errors.New("export needs exactly one output path, or - for stdout")
	}

	db, err := openDB(*dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	where := []string{"1=1"}
	params := []any{}

	if *start != "" {
		s, err := parsePartialStart(*start)
		if err != nil {
			return err
		}
		where = append(where, "dt >= ?")
		params = append(params, s)
	}
	if *stop != "" {
		e, err := parsePartialStop(*stop)
		if err != nil {
			return err
		}
		where = append(where, "dt < ?")
		params = append(params, e)
	}

	query := `SELECT dt, lat, lng, COALESCE(altitude,''), COALESCE(angle,''), COALESCE(speed,''), params
FROM points
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY dt, id`

	rows, err := db.Query(query, params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var w io.Writer
	var closeFn func() error
	outPath := rest[0]
	if outPath == "-" {
		w = os.Stdout
		closeFn = func() error { return nil }
	} else {
		f, err := os.Create(outPath)
		if err != nil {
			return err
		}
		w = f
		closeFn = f.Close
	}
	defer closeFn()

	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{"dt", "lat", "lng", "altitude", "angle", "speed", "params"}); err != nil {
		return err
	}

	for rows.Next() {
		var dt, lat, lng, altitude, angle, speed, params string
		if err := rows.Scan(&dt, &lat, &lng, &altitude, &angle, &speed, &params); err != nil {
			return err
		}
		if err := cw.Write([]string{dt, lat, lng, altitude, angle, speed, params}); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return cw.Error()
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return nil, err
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS points (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,
  format TEXT NOT NULL,
  dt TEXT NOT NULL,
  lat REAL NOT NULL,
  lng REAL NOT NULL,
  altitude REAL,
  angle REAL,
  speed REAL,
  params TEXT NOT NULL DEFAULT '',
  point_hash TEXT NOT NULL UNIQUE,
  imported_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_points_dt ON points(dt);
CREATE INDEX IF NOT EXISTS idx_points_lat_lng ON points(lat, lng);
`); err != nil {
		return nil, err
	}
	return db, nil
}

func columns(header []string) map[string]int {
	out := make(map[string]int, len(header))
	for i, h := range header {
		out[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return out
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func nullable(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func sourceName(path string) string {
	if path == "stdin" {
		return path
	}
	return filepath.Base(path)
}

func (p Point) Hash() string {
	s := strings.Join([]string{p.Format, p.DT, p.Lat, p.Lng, p.Altitude, p.Angle, p.Speed, p.Params}, "|")
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func parsePartialStart(s string) (string, error) {
	t, err := parsePartial(s)
	if err != nil {
		return "", err
	}
	return t.Format(timeLayout), nil
}

func parsePartialStop(s string) (string, error) {
	t, err := parsePartial(s)
	if err != nil {
		return "", err
	}

	s = strings.TrimSpace(s)
	switch len(s) {
	case 4:
		t = t.AddDate(1, 0, 0)
	case 7:
		t = t.AddDate(0, 1, 0)
	case 10:
		t = t.AddDate(0, 0, 1)
	case 19:
		t = t.Add(time.Second)
	default:
		return "", fmt.Errorf("unsupported date/time filter: %q", s)
	}
	return t.Format(timeLayout), nil
}

func parsePartial(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	layouts := []string{
		"2006",
		"2006-01",
		"2006-01-02",
		timeLayout,
	}
	for _, layout := range layouts {
		if len(s) == len(layout) {
			return time.Parse(layout, s)
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date/time filter: %q", s)
}
