package middleware

import (
	"github.com/Frankuccino/gobpt/internal/server"
	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/integrations/nrecho-v4"
	"github.com/newrelic/go-agent/v3/integrations/nrpkgerrors"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type TracingMiddleware struct {
	server *server.Server
	nrApp  *newrelic.Application
}

func NewTracingMiddleware(s *server.Server, nrApp *newrelic.Application) *TracingMiddleware {
	return &TracingMiddleware{
		server: s,
		nrApp:  nrApp,
	}
}

// NewRelicMiddleware returns the New Relic middleware for echo
func (tm *TracingMiddleware) NewRelicMiddleware() echo.MiddlewareFunc {
	if tm.nrApp == nil {
		// return a no-op middleware if New Relic is not initialized
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}
	return nrecho.Middleware(tm.nrApp)
}

// EnhanceTracing adds custom attributes to New Relic transactions
func (tm *TracingMiddleware) EnhanceTracing() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get New Relic transaction from context
			txn := newrelic.FromContext(c.Request().Context())
			if txn == nil {
				return next(c)
			}

			// Add custom attributes
			txn.AddAttribute("service.name", tm.server.Config.Observability.ServiceName)
			txn.AddAttribute("service.environment",
				tm.server.Config.Observability.Environment)
			txn.AddAttribute("http.real_ip", c.RealIP())
			txn.AddAttribute("http.user_agent", c.Request().UserAgent())

			// Add request ID if available
			if requestID := GetRequestID(c); requestID != "" {
				txn.AddAttribute("request_id", requestID)
			}

			// Add user context if available
			if userID := c.Get("user_id"); userID != nil {
				if userIDStr, ok := userID.(string); ok {
					txn.AddAttribute("user_id", userIDStr)
				}
			}

			// Execute next handler
			err := next(c)
			// Record error if any with enhanced stack traces
			if err != nil {
				txn.NoticeError(nrpkgerrors.Wrap(err))
			}

			// Add response status
			txn.AddAttribute("http.status_code", c.Response().Status)

			return err
		}
	}
}

// this is another New Relic integration that will help us with creating our traces
// and adding more data and adding more useful data to our traces to our New Relic transactions
