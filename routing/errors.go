package routing

import "errors"

var (
	ErrNotImplemented      = errors.New("routing: not implemented")
	ErrUnknownProvider     = errors.New("routing: unknown provider")
	ErrProviderUnavailable = errors.New("routing: provider unavailable")
	ErrMissingProviderName = errors.New("routing: missing provider name")
	ErrNilProvider         = errors.New("routing: nil provider")
)
