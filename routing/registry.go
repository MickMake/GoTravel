package routing

import (
	"sort"
)

// Factory builds a Provider instance.
type Factory func() Provider

// Registry stores provider factories by stable name.
type Registry struct {
	factories map[string]Factory
}

func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

func (r *Registry) Register(name string, factory Factory) {
	if r == nil || name == "" || factory == nil {
		return
	}
	r.factories[name] = factory
}

func (r *Registry) Get(name string) (Provider, error) {
	if r == nil {
		return nil, ErrUnknownProvider
	}
	factory, ok := r.factories[name]
	if !ok {
		return nil, ErrUnknownProvider
	}
	return factory(), nil
}

func (r *Registry) Names() []string {
	if r == nil {
		return nil
	}
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
