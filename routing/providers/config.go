package providers

// Config selects and configures a routing provider.
type Config struct {
	Name     string
	ORS      ORSConfig
	OSRM     OSRMConfig
	Valhalla ValhallaConfig
}

// ORSConfig contains the OpenRouteService-specific provider settings.
type ORSConfig struct {
	BaseURL string
	Profile string
}

// OSRMConfig contains the OSRM-specific provider settings.
type OSRMConfig struct {
	BaseURL string
	Profile string
}

// ValhallaConfig contains the Valhalla-specific provider settings.
type ValhallaConfig struct {
	BaseURL string
	Profile string
}
