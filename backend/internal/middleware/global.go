package middleware

import (
	"errors"
	"net/http"

	"github.com/Frankuccino/gobpt/internal/errs"
	"github.com/Frankuccino/gobpt/internal/server"
	"github.com/Frankuccino/gobpt/internal/sqlerr"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

// GlobalMiddlewares bundles application-wide HTTP middleware configurations.
type GlobalMiddlewares struct {
	server *server.Server
}

// NewGlobalMiddlewares initializes and returns a new instanec of GlobalMiddlewares.
func NewGlobalMiddlewares(s *server.Server) *GlobalMiddlewares {
	return &GlobalMiddlewares{
		server: s,
	}
}

// CORS returns an Echo middleware that configures Cross-Origin Resource Sharing
// based on the server's configuration allowed origins.
func (global *GlobalMiddlewares) CORS() echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: global.server.Config.Server.CORSAllowedOrigins,
	})
}

// RequestLogger is a middleware that tries to log every single request to our app
// by returning an Echo middleware that handles structured, contextual logging
// for HTTP requests, adjusting log levels based on response status codes and resolving
// underlying error codes.
func (global *GlobalMiddlewares) RequestLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:     true,
		LogStatus:  true,
		LogError:   true,
		LogLatency: true,
		LogHost:    true,
		LogMethod:  true,
		LogURIPath: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			statusCode := v.Status

			// note that the status code is not yet as it gets picked by the global err handler
			// see here: https://github.com/labstack/echo/issues/2310#issuecomment-1288196898
			if v.Error != nil {
				var httpErr *errs.HTTPError
				var echoErr *echo.HTTPError
				if errors.As(v.Error, &httpErr) {
					statusCode = httpErr.Status
				} else if errors.As(v.Error, &echoErr) {
					statusCode = echoErr.Code
				}

			}

			// Get enhanced logger from context
			logger := GetLogger(c)

			var e *zerolog.Event

			switch {
			case statusCode >= 500:
				e = logger.Error().Err(v.Error)
			case statusCode >= 400:
				e = logger.Warn()
			default:
				e = logger.Info()
			}

			// Add request ID if available
			if requestID := GetRequestID(c); requestID != "" {
				e = e.Str("user_id", requestID)
			}

			// Add user context if available
			if userID := GetUserID(c); userID != "" {
				e = e.Str("user_id", userID)
			}

			e.
				Dur("latency", v.Latency).
				Int("status", statusCode).
				Str("method", v.Method).
				Str("uri", v.URI).
				Str("host", v.Host).
				Str("ip", c.RealIP()).
				Str("user_agent", c.Request().UserAgent()).
				Msg("API")

			return nil
		},
	})
}

// Recover returns an Echo middleware that recovers from panics anywhere in the chain,
// logs the panic, and handles it gracefully.
func (global *GlobalMiddlewares) Recover() echo.MiddlewareFunc {
	return middleware.Recover()
}

// Secure return an Echo middleware that sets variious HTTP headers for basic security protections.
func (global *GlobalMiddlewares) Secure() echo.MiddlewareFunc {
	return middleware.Secure()
}

// GlobalErrorHandler is a centralized custom error handler for Echo that intercepts runtime errors,
// translates low-level database constraints via the sqlerr package into standardized HTTP errors,
// logs failures with full stack traces, and responds with a uniform JSON error payload.
func (global *GlobalMiddlewares) GlobalErrorHandler(err error, c echo.Context) {
	// First try to handle database errors and convert them to appropriate HTTP errors
	originalErr := err

	// Try to handle known database errors
	// Only do this for errors that haven't already been converted to HTTPError
	var httpErr *errs.HTTPError
	if !errors.As(err, &httpErr) {
		var echoErr *echo.HTTPError
		if errors.As(err, &echoErr) {
			if echoErr.Code == http.StatusNotFound {
				err = errs.NewNotFoundError("Route not found", false, nil)
			}
		} else {
			// Here we call our sqlerr handler which will convert database errors
			// to appropriate application errors
			err = sqlerr.HandleError(err)
		}
	}

	// Now process the possible converted error
	var echoErr *echo.HTTPError
	var status int
	var code string
	var message string
	var fieldErrors []errs.FieldError
	var action *errs.Action

	switch {
	case errors.As(err, &httpErr):
		status = httpErr.Status
		code = httpErr.Code
		message = httpErr.Message
		fieldErrors = httpErr.Errors
		action = httpErr.Action

	case errors.As(err, &echoErr):
		status = echoErr.Code
		code = errs.MakeUpperCaseWithUnderscores(http.StatusText(status))
		if msg, ok := echoErr.Message.(string); ok {
			message = msg
		} else {
			message = http.StatusText(echoErr.Code)
		}

	default:
		status = http.StatusInternalServerError
		code = errs.MakeUpperCaseWithUnderscores(
			http.StatusText(http.StatusInternalServerError))
		message = http.StatusText(http.StatusInternalServerError)
	}

	// Log the original error to help with debugging
	// Use enhanced logger from context which already includes request_id, method, path, ip, user context, and trace context
	logger := *GetLogger(c)

	logger.Error().Stack().
		Err(originalErr).
		Int("status", status).
		Str("error_code", code).
		Msg(message)

	if !c.Response().Committed {
		_ = c.JSON(status, errs.HTTPError{
			Code:     code,
			Message:  message,
			Status:   status,
			Override: httpErr != nil && httpErr.Override,
			Errors:   fieldErrors,
			Action:   action,
		})
	}
}

// for this package to work, we also have to create another package for dealing with database-based errors.
