package providers

// Config selects and configures a routing provider.
type Config struct {
	Name     string
	OSRM     OSRMConfig
	Valhalla ValhallaConfig
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
