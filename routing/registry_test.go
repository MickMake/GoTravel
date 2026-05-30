package routing_test

import (
	"errors"
	"testing"

	"github.com/MickMake/GoTravel/routing"
	"github.com/MickMake/GoTravel/routing/noop"
	"github.com/MickMake/GoTravel/routing/ors"
	"github.com/MickMake/GoTravel/routing/osrm"
	"github.com/MickMake/GoTravel/routing/valhalla"
)

func TestRegistryNamesSorted(t *testing.T) {
	r := routing.NewRegistry()
	r.Register(osrm.Name, func() routing.Provider { return osrm.New() })
	r.Register(noop.Name, func() routing.Provider { return noop.New() })
	r.Register(valhalla.Name, func() routing.Provider { return valhalla.New() })
	r.Register(ors.Name, func() routing.Provider { return ors.New() })

	got := r.Names()
	want := []string{"noop", "ors", "osrm", "valhalla"}
	if len(got) != len(want) {
		t.Fatalf("len(names)=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("names[%d]=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestRegistryZeroValueRegister(t *testing.T) {
	var r routing.Registry
	r.Register(noop.Name, func() routing.Provider { return noop.New() })

	p, err := r.Get(noop.Name)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Name() != noop.Name {
		t.Fatalf("name=%q want=%q", p.Name(), noop.Name)
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	r := routing.NewRegistry()
	_, err := r.Get("missing")
	if !errors.Is(err, routing.ErrUnknownProvider) {
		t.Fatalf("expected ErrUnknownProvider, got %v", err)
	}
}

func TestRegistryGetKnown(t *testing.T) {
	r := routing.NewRegistry()
	r.Register(noop.Name, func() routing.Provider { return noop.New() })

	p, err := r.Get(noop.Name)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Name() != noop.Name {
		t.Fatalf("name=%q want=%q", p.Name(), noop.Name)
	}
}
