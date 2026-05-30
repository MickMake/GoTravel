package providers

import (
	"strings"

	"github.com/MickMake/GoTravel/routing"
	"github.com/MickMake/GoTravel/routing/noop"
	"github.com/MickMake/GoTravel/routing/ors"
	"github.com/MickMake/GoTravel/routing/osrm"
	"github.com/MickMake/GoTravel/routing/valhalla"
)

func Names() []string {
	return []string{"noop", "ors", "osrm", "valhalla"}
}

func New(config Config) (routing.Provider, error) {
	name := strings.ToLower(strings.TrimSpace(config.Name))
	if name == "" {
		return nil, routing.ErrMissingProviderName
	}

	switch name {
	case "noop":
		return noop.New(), nil
	case "ors":
		return ors.New(), nil
	case "osrm":
		return osrm.NewWithConfig(osrm.Config{BaseURL: config.OSRM.BaseURL, Profile: config.OSRM.Profile}), nil
	case "valhalla":
		return valhalla.NewWithConfig(valhalla.Config{BaseURL: config.Valhalla.BaseURL, Profile: config.Valhalla.Profile}), nil
	default:
		return nil, routing.ErrUnknownProvider
	}
}
