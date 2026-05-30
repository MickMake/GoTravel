package routing_test

import (
	"context"
	"fmt"

	"github.com/MickMake/GoTravel/routing"
)

func ExampleEnricher_MatchTrace() {
	provider := exampleProvider{name: "example"}
	service, err := routing.NewService(provider)
	if err != nil {
		panic(err)
	}
	enricher, err := routing.NewEnricher(service)
	if err != nil {
		panic(err)
	}

	result, err := enricher.MatchTrace(context.Background(), routing.MatchTraceRequest{Profile: "driving"})
	if err != nil {
		panic(err)
	}

	fmt.Println(result.Provider)
	fmt.Println(result.Profile)

	// Output:
	// example
	// driving
}

type exampleProvider struct {
	name string
}

func (p exampleProvider) Name() string                     { return p.name }
func (p exampleProvider) Health(ctx context.Context) error { return nil }
func (p exampleProvider) Capabilities(ctx context.Context) routing.Capabilities {
	return routing.Capabilities{MatchTrace: true}
}
func (p exampleProvider) Route(ctx context.Context, req routing.RouteRequest) (routing.RouteResult, error) {
	return routing.RouteResult{}, routing.ErrNotImplemented
}
func (p exampleProvider) MatchTrace(ctx context.Context, req routing.MatchTraceRequest) (routing.MatchTraceResult, error) {
	return routing.MatchTraceResult{Provider: p.name, Profile: req.Profile, Status: "Ok"}, nil
}
func (p exampleProvider) Snap(ctx context.Context, req routing.SnapRequest) (routing.SnapResult, error) {
	return routing.SnapResult{}, routing.ErrNotImplemented
}
func (p exampleProvider) Matrix(ctx context.Context, req routing.MatrixRequest) (routing.MatrixResult, error) {
	return routing.MatrixResult{}, routing.ErrNotImplemented
}
