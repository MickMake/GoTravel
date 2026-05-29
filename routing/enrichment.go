package routing

import "context"

// Enricher is the provider-neutral entry point for routing enrichment operations.
type Enricher struct {
	service *Service
}

// NewEnricher creates an Enricher backed by service.
func NewEnricher(service *Service) (*Enricher, error) {
	if service == nil {
		return nil, ErrNilService
	}
	return &Enricher{service: service}, nil
}

// MatchTrace asks the configured routing service to match a GPS trace.
func (e *Enricher) MatchTrace(ctx context.Context, req MatchTraceRequest) (MatchTraceResult, error) {
	return e.service.MatchTrace(ctx, req)
}
