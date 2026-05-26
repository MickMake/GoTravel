package valhalla

import (
	"context"

	"github.com/MickMake/GoTravel/routing"
)

const Name = "valhalla"

type Provider struct{}

func New() *Provider                                 { return &Provider{} }
func (p *Provider) Name() string                     { return Name }
func (p *Provider) Health(ctx context.Context) error { return routing.ErrNotImplemented }
func (p *Provider) Capabilities(ctx context.Context) routing.Capabilities {
	return routing.Capabilities{}
}
func (p *Provider) Route(ctx context.Context, req routing.RouteRequest) (routing.RouteResult, error) {
	return routing.RouteResult{}, routing.ErrNotImplemented
}
func (p *Provider) MatchTrace(ctx context.Context, req routing.MatchTraceRequest) (routing.MatchTraceResult, error) {
	return routing.MatchTraceResult{}, routing.ErrNotImplemented
}
func (p *Provider) Snap(ctx context.Context, req routing.SnapRequest) (routing.SnapResult, error) {
	return routing.SnapResult{}, routing.ErrNotImplemented
}
func (p *Provider) Matrix(ctx context.Context, req routing.MatrixRequest) (routing.MatrixResult, error) {
	return routing.MatrixResult{}, routing.ErrNotImplemented
}
