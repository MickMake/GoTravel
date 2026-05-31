package storage

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var requiredTables = []string{"points", "import_runs", "import_errors"}

func InitDatabase(path string, force bool) error {
	if path == "" {
		return fmt.Errorf("database path is required")
	}
	if force {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	store, err := Open(path)
	if err != nil {
		return err
	}
	return store.Close()
}

func ExportDatabase(sourcePath, outputPath string, force bool) error {
	if sourcePath == "" {
		return fmt.Errorf("database path is required")
	}
	if outputPath == "" {
		return fmt.Errorf("output filename is required")
	}
	if outputPath == "-" {
		return fmt.Errorf("db export requires a filename, not stdout")
	}
	if samePath(sourcePath, outputPath) {
		return fmt.Errorf("db export source and output must be different files")
	}
	if err := ValidateDatabase(sourcePath); err != nil {
		return err
	}
	return copyFile(sourcePath, outputPath, force)
}

func ImportDatabase(inputPath, targetPath string, force bool) error {
	if inputPath == "" {
		return fmt.Errorf("input filename is required")
	}
	if targetPath == "" {
		return fmt.Errorf("database path is required")
	}
	if samePath(inputPath, targetPath) {
		return fmt.Errorf("db import input and target must be different files")
	}
	if err := ValidateDatabase(inputPath); err != nil {
		return err
	}
	if _, err := os.Stat(targetPath); err == nil && !force {
		return fmt.Errorf("database %q already exists; use --force to overwrite", targetPath)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	dir := filepath.Dir(targetPath)
	tmp, err := os.CreateTemp(dir, ".gotravel-import-*.sqlite")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	defer os.Remove(tmpPath)

	if err := copyFile(inputPath, tmpPath, true); err != nil {
		return err
	}
	if err := ValidateDatabase(tmpPath); err != nil {
		return err
	}
	if force {
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return os.Rename(tmpPath, targetPath)
}

func ValidateDatabase(path string) error {
	if path == "" {
		return fmt.Errorf("database path is required")
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("database %q does not exist", path)
		}
		return err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("database %q is not a usable SQLite database: %w", path, err)
	}
	for _, table := range requiredTables {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err == sql.ErrNoRows {
			return fmt.Errorf("database %q is missing required table %q", path, table)
		}
		if err != nil {
			return fmt.Errorf("database %q is not a usable GoTravel database: %w", path, err)
		}
	}
	return nil
}

func copyFile(sourcePath, outputPath string, force bool) error {
	in, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := OpenOutputFile(outputPath, force)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil {
		return absA == absB
	}
	return a == b
}
