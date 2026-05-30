package routing

import "context"

// Service provides a small provider-neutral wrapper around a routing provider.
type Service struct {
	provider Provider
}

// NewService creates a Service backed by provider.
func NewService(provider Provider) (*Service, error) {
	if provider == nil {
		return nil, ErrNilProvider
	}
	return &Service{provider: provider}, nil
}

// Provider returns the provider used by the service.
func (s *Service) Provider() Provider {
	if s == nil {
		return nil
	}
	return s.provider
}

func (s *Service) Health(ctx context.Context) error {
	return s.provider.Health(ctx)
}

func (s *Service) Capabilities(ctx context.Context) Capabilities {
	return s.provider.Capabilities(ctx)
}

func (s *Service) Route(ctx context.Context, req RouteRequest) (RouteResult, error) {
	return s.provider.Route(ctx, req)
}

func (s *Service) MatchTrace(ctx context.Context, req MatchTraceRequest) (MatchTraceResult, error) {
	return s.provider.MatchTrace(ctx, req)
}

func (s *Service) Snap(ctx context.Context, req SnapRequest) (SnapResult, error) {
	return s.provider.Snap(ctx, req)
}

func (s *Service) Matrix(ctx context.Context, req MatrixRequest) (MatrixResult, error) {
	return s.provider.Matrix(ctx, req)
}
