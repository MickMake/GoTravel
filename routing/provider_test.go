package routing_test

import (
	"testing"

	"github.com/MickMake/GoTravel/routing"
	"github.com/MickMake/GoTravel/routing/noop"
	"github.com/MickMake/GoTravel/routing/ors"
	"github.com/MickMake/GoTravel/routing/osrm"
	"github.com/MickMake/GoTravel/routing/valhalla"
)

func TestProviderNamesStable(t *testing.T) {
	tests := []struct {
		name string
		p    routing.Provider
		want string
	}{
		{name: "noop", p: noop.New(), want: noop.Name},
		{name: "ors", p: ors.New(), want: ors.Name},
		{name: "osrm", p: osrm.New(), want: osrm.Name},
		{name: "valhalla", p: valhalla.New(), want: valhalla.Name},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Name(); got != tt.want {
				t.Fatalf("Name()=%q want=%q", got, tt.want)
			}
		})
	}
}
