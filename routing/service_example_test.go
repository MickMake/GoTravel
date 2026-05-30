package routing_test

import (
	"fmt"

	"github.com/MickMake/GoTravel/routing"
	"github.com/MickMake/GoTravel/routing/noop"
)

func ExampleNewService() {
	provider := noop.New()
	service, err := routing.NewService(provider)
	if err != nil {
		panic(err)
	}

	fmt.Println(service.Provider().Name())

	// Output:
	// noop
}
