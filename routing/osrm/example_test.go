package osrm

import (
	"fmt"

	"github.com/MickMake/GoTravel/routing"
)

var _ routing.Provider = (*Provider)(nil)

func ExampleNew() {
	provider := New()

	fmt.Println(provider.Name())

	// Output:
	// osrm
}

func ExampleNewWithConfig() {
	provider := NewWithConfig(Config{
		BaseURL: "http://127.0.0.1:5000",
		Profile: "driving",
	})

	fmt.Println(provider.Name())

	// Output:
	// osrm
}
