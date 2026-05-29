package routing

import (
	"strings"
	"testing"
	"time"
)

func TestEnrichedTraceFromMatchTraceResult(t *testing.T) {
	confidence := 0.91
	matchedAt := time.Unix(1700000100, 0)
	result := MatchTraceResult{
		Provider:        "osrm",
		Profile:         "driving",
		Status:          "Ok",
		Geometry:        "encoded",
		GeometryFormat:  "polyline6",
		DistanceMeters:  123.4,
		DurationSeconds: 56.7,
		Confidence:      &confidence,
		Warnings:        []string{"minor warning"},
		RawResponse:     []byte(`{"code":"Ok"}`),
	}

	enriched, err := EnrichedTraceFromMatchTraceResult(result, 2, matchedAt)
	if err != nil {
		t.Fatalf("EnrichedTraceFromMatchTraceResult() err=%v", err)
	}
	if enriched.Provider != "osrm" || enriched.Profile != "driving" || enriched.Status != "Ok" {
		t.Fatalf("unexpected identity/status: %+v", enriched)
	}
	if enriched.SourcePointCount != 2 || enriched.Geometry != "encoded" || enriched.GeometryFormat != "polyline6" {
		t.Fatalf("unexpected trace metadata: %+v", enriched)
	}
	if enriched.DistanceMeters != 123.4 || enriched.DurationSeconds != 56.7 {
		t.Fatalf("unexpected metrics: %+v", enriched)
	}
	if enriched.Confidence == nil || *enriched.Confidence != confidence {
		t.Fatalf("confidence=%v", enriched.Confidence)
	}
	if len(enriched.Warnings) != 1 || enriched.Warnings[0] != "minor warning" {
		t.Fatalf("warnings=%+v", enriched.Warnings)
	}
	if string(enriched.RawResponse) != `{"code":"Ok"}` || !enriched.MatchedAt.Equal(matchedAt) {
		t.Fatalf("unexpected raw/matched time: %+v", enriched)
	}
}

func TestEnrichedTraceFromMatchTraceResultCopiesMutableFields(t *testing.T) {
	confidence := 0.5
	result := MatchTraceResult{
		Confidence:  &confidence,
		Warnings:    []string{"original"},
		RawResponse: []byte("original"),
	}

	enriched, err := EnrichedTraceFromMatchTraceResult(result, 2, time.Unix(1700000100, 0))
	if err != nil {
		t.Fatalf("EnrichedTraceFromMatchTraceResult() err=%v", err)
	}

	confidence = 0.1
	result.Warnings[0] = "mutated"
	result.RawResponse[0] = 'X'

	if enriched.Confidence == nil || *enriched.Confidence != 0.5 {
		t.Fatalf("confidence was not copied: %v", enriched.Confidence)
	}
	if enriched.Warnings[0] != "original" {
		t.Fatalf("warnings were not copied: %+v", enriched.Warnings)
	}
	if string(enriched.RawResponse) != "original" {
		t.Fatalf("raw response was not copied: %s", string(enriched.RawResponse))
	}
}

func TestEnrichedTraceFromMatchTraceResultRejectsInvalidInputs(t *testing.T) {
	_, err := EnrichedTraceFromMatchTraceResult(MatchTraceResult{}, 1, time.Unix(1700000100, 0))
	if err == nil || !strings.Contains(err.Error(), "at least two source points") {
		t.Fatalf("source point err=%v", err)
	}

	_, err = EnrichedTraceFromMatchTraceResult(MatchTraceResult{}, 2, time.Time{})
	if err == nil || !strings.Contains(err.Error(), "matched timestamp") {
		t.Fatalf("matched timestamp err=%v", err)
	}
}
