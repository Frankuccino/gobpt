package middleware

import (
	"github.com/Frankuccino/gobpt/internal/server"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// Middlewares acts as a centralized container holder all application-level
// middleware components (Global, Auth, ContextEnhancer, Tracing, and RateLimit)
type Middlewares struct {
	Global          *GlobalMiddlewares
	Auth            *AuthMiddleware
	ContextEnhancer *ContextEnhancer
	Tracing         *TracingMiddleware
	RateLimit       *RateLimitMiddleware
}

// NewMiddleware initializes and returns a unified Middlewares instance,
// resolving dependencies like the New Relic application tracer from the server configuration.
func NewMiddlewares(s *server.Server) *Middlewares {
	// Get New Relic application instance from server
	var nrApp *newrelic.Application
	if s.LoggerService != nil {
		nrApp = s.LoggerService.GetApplication()
	}

	return &Middlewares{
		Global:          NewGlobalMiddlewares(s),
		Auth:            NewAuthMiddleware(s),
		ContextEnhancer: NewContextEnhancer(s),
		Tracing:         NewTracingMiddleware(s, nrApp),
		RateLimit:       NewRateLimitMiddleware(s),
	}
}
