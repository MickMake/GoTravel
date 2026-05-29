package providers

// Config selects and configures a routing provider.
type Config struct {
	Name string
	OSRM OSRMConfig
}

// OSRMConfig contains the OSRM-specific provider settings.
type OSRMConfig struct {
	BaseURL string
	Profile string
}
