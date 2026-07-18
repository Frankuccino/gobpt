package middleware

import (
	"github.com/Frankuccino/gobpt/internal/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type GlobalMiddlewares struct {
	server *server.Server
}

func NewGlobalMiddlewares(s *sever.Server) *GlobalMiddlewares {
	return &GlobalMiddlewares{
		server: s,
	}
}

func (global *GlobalMiddlewares) CORS() echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: global.server.Config.Server.CORSAllowedOrigins,
	})
}

func (global *GlobalMiddlewares) RequestLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogError:  true,
	})
}

// for this package to work, we also have to create another package for dealing with database-based errors.
