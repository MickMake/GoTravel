package providers_test

import (
	"fmt"

	"github.com/MickMake/GoTravel/routing"
	"github.com/MickMake/GoTravel/routing/providers"
)

func ExampleNew() {
	provider, err := providers.New(providers.Config{Name: "osrm"})
	if err != nil {
		panic(err)
	}
	service, err := routing.NewService(provider)
	if err != nil {
		panic(err)
	}

	fmt.Println(service.Provider().Name())

	// Output:
	// osrm
}

func ExampleNew_withOSRMConfig() {
	provider, err := providers.New(providers.Config{
		Name: "osrm",
		OSRM: providers.OSRMConfig{
			BaseURL: "http://127.0.0.1:5000",
			Profile: "driving",
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(provider.Name())

	// Output:
	// osrm
}
