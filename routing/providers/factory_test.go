package providers

import (
	"errors"
	"reflect"
	"testing"

	"github.com/MickMake/GoTravel/routing"
)

func TestNamesReturnsStableProviderNames(t *testing.T) {
	want := []string{"noop", "ors", "osrm", "valhalla"}
	if got := Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names()=%v want %v", got, want)
	}
}

func TestNewCreatesBuiltInProviders(t *testing.T) {
	for _, name := range Names() {
		provider, err := New(Config{Name: name})
		if err != nil {
			t.Fatalf("New(%q) err=%v", name, err)
		}
		if provider.Name() != name {
			t.Fatalf("New(%q).Name()=%q", name, provider.Name())
		}
	}
}

func TestNewNormalisesProviderName(t *testing.T) {
	provider, err := New(Config{Name: " OSRM "})
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}
	if provider.Name() != "osrm" {
		t.Fatalf("Name()=%q want osrm", provider.Name())
	}
}

func TestNewWiresORSConfig(t *testing.T) {
	provider, err := New(Config{Name: "ors", ORS: ORSConfig{BaseURL: "http://127.0.0.1:1234/ors", Profile: "foot-walking"}})
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}
	result, err := provider.MatchTrace(nil, routing.MatchTraceRequest{})
	if !errors.Is(err, routing.ErrNotImplemented) {
		t.Fatalf("MatchTrace() err=%v want ErrNotImplemented", err)
	}
	if result.Profile != "foot-walking" {
		t.Fatalf("profile=%q want foot-walking", result.Profile)
	}
}

func TestNewRejectsMissingProviderName(t *testing.T) {
	provider, err := New(Config{})
	if provider != nil {
		t.Fatalf("provider=%v want nil", provider)
	}
	if !errors.Is(err, routing.ErrMissingProviderName) {
		t.Fatalf("err=%v want ErrMissingProviderName", err)
	}
}

func TestNewRejectsUnknownProvider(t *testing.T) {
	provider, err := New(Config{Name: "badger"})
	if provider != nil {
		t.Fatalf("provider=%v want nil", provider)
	}
	if !errors.Is(err, routing.ErrUnknownProvider) {
		t.Fatalf("err=%v want ErrUnknownProvider", err)
	}
}
