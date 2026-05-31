package storage

import (
	"path/filepath"
	"testing"
)

func TestTripSchemaInitAndValidate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "gotravel.sqlite")
	if err := InitDatabase(dbPath, false); err != nil {
		t.Fatalf("InitDatabase() err=%v", err)
	}
	if err := ValidateDatabase(dbPath); err != nil {
		t.Fatalf("ValidateDatabase() err=%v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() err=%v", err)
	}
	defer store.Close()
	for _, table := range []string{"trips", "trip_points"} {
		var name string
		if err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("missing table %s: %v", table, err)
		}
		if name != table {
			t.Fatalf("table name=%q want %q", name, table)
		}
	}
}
