package noop

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MickMake/GoTravel/routing"
)

const Name = "noop"

type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) Name() string                     { return Name }
func (p *Provider) Health(ctx context.Context) error { return nil }
func (p *Provider) Capabilities(ctx context.Context) routing.Capabilities {
	return routing.Capabilities{MatchTrace: true}
}
func (p *Provider) Route(ctx context.Context, req routing.RouteRequest) (routing.RouteResult, error) {
	return routing.RouteResult{}, routing.ErrNotImplemented
}
func (p *Provider) MatchTrace(ctx context.Context, req routing.MatchTraceRequest) (routing.MatchTraceResult, error) {
	coordinates := make([][]float64, 0, len(req.Points))
	for _, point := range req.Points {
		coordinates = append(coordinates, []float64{point.Lng, point.Lat})
	}
	geometry, err := json.Marshal(map[string]any{
		"type":        "LineString",
		"coordinates": coordinates,
	})
	if err != nil {
		return routing.MatchTraceResult{}, fmt.Errorf("noop match trace: encode geometry: %w", err)
	}
	return routing.MatchTraceResult{
		Provider:       Name,
		Profile:        req.Profile,
		Status:         "ok",
		Geometry:       string(geometry),
		GeometryFormat: "geojson",
		RawResponse:    []byte(`{"provider":"noop","status":"ok"}`),
	}, nil
}
func (p *Provider) Snap(ctx context.Context, req routing.SnapRequest) (routing.SnapResult, error) {
	return routing.SnapResult{}, routing.ErrNotImplemented
}
func (p *Provider) Matrix(ctx context.Context, req routing.MatrixRequest) (routing.MatrixResult, error) {
	return routing.MatrixResult{}, routing.ErrNotImplemented
}
