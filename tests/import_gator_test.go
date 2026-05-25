package tests

import (
	"os"
	"testing"

	importers "github.com/MickMake/GoTravel/import"
)

func TestGatorImportValid(t *testing.T) {
	f, err := os.Open("fixtures/gator_sample.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	imp, err := importers.New("gator")
	if err != nil {
		t.Fatal(err)
	}
	result := imp.Import(f, "fixtures/gator_sample.csv")
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(result.Errors))
	}
	if len(result.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(result.Points))
	}
}

func TestGatorImportCorrupt(t *testing.T) {
	f, err := os.Open("fixtures/gator_corrupt.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	imp, err := importers.New("gator")
	if err != nil {
		t.Fatal(err)
	}
	result := imp.Import(f, "fixtures/gator_corrupt.csv")
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
}
