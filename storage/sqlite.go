package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const timeLayout = "2006-01-02 15:04:05"

type DB struct {
	db *sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &DB{db: db}
	if err := store.enableForeignKeys(); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.EnsureSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *DB) Close() error { return s.db.Close() }

func (s *DB) enableForeignKeys() error {
	_, err := s.db.Exec(`PRAGMA foreign_keys = ON`)
	return err
}

func (s *DB) EnsureSchema() error {
	_, err := s.db.Exec(schemaSQL)
	return err
}

func PointHash(p Point) string {
	parts := []string{
		p.DT.Format(timeLayout),
		fmt.Sprintf("%.7f", p.Lat),
		fmt.Sprintf("%.7f", p.Lng),
		fmt.Sprintf("%.3f", p.Altitude),
		fmt.Sprintf("%.3f", p.Angle),
		fmt.Sprintf("%.3f", p.Speed),
		strings.TrimSpace(p.Params),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func (s *DB) SaveImportResult(result ImportResult, force bool) (imported int, skipped int, err error) {
	if len(result.Errors) > 0 && !force {
		first := result.Errors[0]
		return 0, 0, CorruptInputError{SourceFile: first.SourceFile, Line: first.SourceLine, Reason: first.Error}
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().UTC().Format(timeLayout)
	res, err := tx.ExecContext(ctx, `INSERT INTO import_runs (format, source_file, started_at, status) VALUES (?, ?, ?, ?)`, result.Format, result.SourceFile, now, "running")
	if err != nil {
		return 0, 0, err
	}
	runID, err := res.LastInsertId()
	if err != nil {
		return 0, 0, err
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT OR IGNORE INTO points (dt, lat, lng, altitude, angle, speed, params, format, source_file, source_line, imported_at, point_hash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, 0, err
	}
	defer stmt.Close()

	for _, p := range result.Points {
		p.ImportedAt = time.Now().UTC()
		p.PointHash = PointHash(p)
		insertRes, err := stmt.ExecContext(ctx,
			p.DT.Format(timeLayout), p.Lat, p.Lng, p.Altitude, p.Angle, p.Speed, p.Params,
			p.Format, p.SourceFile, p.SourceLine, p.ImportedAt.Format(timeLayout), p.PointHash,
		)
		if err != nil {
			return imported, skipped, err
		}
		rows, _ := insertRes.RowsAffected()
		if rows > 0 {
			imported++
		} else {
			skipped++
		}
	}

	for _, e := range result.Errors {
		_, err = tx.ExecContext(ctx, `INSERT INTO import_errors (import_run_id, source_file, source_line, format, raw_row, error, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, runID, e.SourceFile, e.SourceLine, e.Format, e.RawRow, e.Error, now)
		if err != nil {
			return imported, skipped, err
		}
	}
	skipped += len(result.Errors)

	status := "ok"
	if len(result.Errors) > 0 {
		status = "completed_with_errors"
	}
	_, err = tx.ExecContext(ctx, `UPDATE import_runs SET finished_at = ?, rows_seen = ?, rows_imported = ?, rows_skipped = ?, status = ? WHERE id = ?`, now, result.RowsSeen, imported, skipped, status, runID)
	if err != nil {
		return imported, skipped, err
	}

	if err = tx.Commit(); err != nil {
		return imported, skipped, err
	}
	return imported, skipped, nil
}

func (s *DB) QueryPoints(start, stop *time.Time) ([]Point, error) {
	query := `SELECT id, dt, lat, lng, altitude, angle, speed, params, format, source_file, source_line, imported_at, point_hash FROM points`
	var args []any
	var where []string
	if start != nil {
		where = append(where, "dt >= ?")
		args = append(args, start.Format(timeLayout))
	}
	if stop != nil {
		where = append(where, "dt <= ?")
		args = append(args, stop.Format(timeLayout))
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY dt, id"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []Point
	for rows.Next() {
		var p Point
		var dt, importedAt string
		if err := rows.Scan(&p.ID, &dt, &p.Lat, &p.Lng, &p.Altitude, &p.Angle, &p.Speed, &p.Params, &p.Format, &p.SourceFile, &p.SourceLine, &importedAt, &p.PointHash); err != nil {
			return nil, err
		}
		p.DT, _ = time.ParseInLocation(timeLayout, dt, time.Local)
		p.ImportedAt, _ = time.ParseInLocation(timeLayout, importedAt, time.Local)
		points = append(points, p)
	}
	return points, rows.Err()
}

func ParsePartialDateTime(value string, isStop bool) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	layouts := []struct {
		layout string
		add    func(time.Time) time.Time
	}{
		{"2006", func(t time.Time) time.Time { return t.AddDate(1, 0, 0).Add(-time.Second) }},
		{"2006-01", func(t time.Time) time.Time { return t.AddDate(0, 1, 0).Add(-time.Second) }},
		{"2006-01-02", func(t time.Time) time.Time { return t.AddDate(0, 0, 1).Add(-time.Second) }},
		{"2006-01-02 15", func(t time.Time) time.Time { return t.Add(time.Hour).Add(-time.Second) }},
		{"2006-01-02 15:04", func(t time.Time) time.Time { return t.Add(time.Minute).Add(-time.Second) }},
		{timeLayout, func(t time.Time) time.Time { return t }},
	}
	for _, item := range layouts {
		if t, err := time.ParseInLocation(item.layout, value, time.Local); err == nil {
			if isStop {
				t = item.add(t)
			}
			return &t, nil
		}
	}
	return nil, fmt.Errorf("invalid date/time %q; expected YYYY, YYYY-MM, YYYY-MM-DD, YYYY-MM-DD HH, YYYY-MM-DD HH:MM, or YYYY-MM-DD HH:MM:SS", value)
}
