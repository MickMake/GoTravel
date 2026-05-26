package routing

import "context"

// Provider defines the provider-neutral routing contract shared by all routing adapters.
type Provider interface {
	Name() string
	Health(ctx context.Context) error
	Capabilities(ctx context.Context) Capabilities

	Route(ctx context.Context, req RouteRequest) (RouteResult, error)
	MatchTrace(ctx context.Context, req MatchTraceRequest) (MatchTraceResult, error)
	Snap(ctx context.Context, req SnapRequest) (SnapResult, error)
	Matrix(ctx context.Context, req MatrixRequest) (MatrixResult, error)
}
