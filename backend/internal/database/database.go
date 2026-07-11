package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/Frankuccino/gobpt/internal/config"
	pgxzero "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/newrelic/go-agent/v3/integrations/nrpgx5"
	"github.com/rs/zerolog"

	loggerConfig "github.com/Frankuccino/gobpt/internal/logger"
)

type Database struct {
	Pool *pgxpool.Pool
	log  *zerolog.Logger
}

// multiTracer allows chaining multiple tracers
type multiTracer struct {
	tracers []any
}

const DatabasePingTimeout = 10

func (mt *multiTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	for _, tracer := range mt.tracers {
		if t, ok := tracer.(interface {
			TraceQueryStart(context.Context, *pgx.Conn, pgx.TraceQueryStartData) context.Context
		}); ok {
			ctx = t.TraceQueryStart(ctx, conn, data)
		}
	}
	return ctx
}

func (mt *multiTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	for _, tracer := range mt.tracers {
		if t, ok := tracer.(interface {
			TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData)
		}); ok {
			t.TraceQueryEnd(ctx, conn, data)
		}
	}
}

// Function that actually create database connection; it'll have our loggers, and tracers integrated in it.
func New(cfg *config.Config, logger *zerolog.Logger, loggerService *loggerConfig.LoggerService) (*Database, error) {
	hostPort := net.JoinHostPort(cfg.Database.Host, strconv.Itoa(cfg.Database.Port))

	// URL-encode the password
	encodedPassword := url.QueryEscape(cfg.Database.Password)
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.Database.User,
		encodedPassword,
		hostPort,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	pgxPoolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgx pool config %w", err)
	}

	// Add New Relic PostgreSQL instrumentation
	if loggerService != nil && loggerService.GetApplication() != nil {
		pgxPoolConfig.ConnConfig.Tracer = nrpgx5.NewTracer()
	}

	// This is where we define the logger either to be a local or production
	if cfg.Primary.Env == "local" {
		globalLevel := logger.GetLevel()
		pgxLogger := loggerConfig.NewPgxLogger(globalLevel)
		// Chain tracers - New Relic first, then local logging
		if pgxPoolConfig.ConnConfig.Tracer != nil {
			// If New Relic tracer exists, create a multi-tracer
			localTracer := &tracelog.TraceLog{
				Logger:   pgxzero.NewLogger(pgxLogger),
				LogLevel: tracelog.LogLevel(loggerConfig.GetPgxTraceLogLevel(globalLevel)),
			}
			pgxPoolConfig.ConnConfig.Tracer = &multiTracer{
				tracers: []any{pgxPoolConfig.ConnConfig.Tracer, localTracer},
			}
		} else {
			pgxPoolConfig.ConnConfig.Tracer = &tracelog.TraceLog{
				Logger:   pgxzero.NewLogger(pgxLogger),
				LogLevel: tracelog.LogLevel(loggerConfig.GetPgxTraceLogLevel(globalLevel)),
			}
		}
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), pgxPoolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	database := &Database{
		Pool: pool,
		log:  logger,
	}

	// Check whether we're able to connect to our database or not.
	ctx, cancel := context.WithTimeout(context.Background(), DatabasePingTimeout*time.Second)
	defer cancel()
	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().Msg("connected to the database")

	return database, nil
}

func (db *Database) Close() error {
	db.log.Info().Msg("closing database connection pool")
	db.Pool.Close()
	return nil
}

// note for self, as you can see from the above struct 'mtTracers', the defined field of 'tracers'
// is a type of an empty interface slice, due to how the usage of 'any' is implicitly an empty interface.

// When we defined a looping condition over the receiver of the methods; we have a struct pointer receiver.
// Specifically, looping over the Field 'tracers' of that struct, which initially is an empty interface.

// We then go ahead and do some conditional statement to produce a Type Assertion to an Anonymous Interface.
// This in turn creates a runtime interface discovery, where it allows the multiTracer to be loosely coupled.
// Preventing "fat interfaces".

// That means, we are defining the method of the interface 'tracers' slice directly through type assertions.
// We Type asserted an Interface with the method(s) we defined and their data types.
// This allows us to produce that runtime interface discovery, since we haven't defined an interface explicitly
// or in the context of a compile-time definition.

// And as we know, type assertions; it is a two-value expression, extracting the value, and boolean value.
// So it's denoted with the underlying value itself, and the value of true/false,
// we get a 'true' value when type asserting is successful on the 2nd value, usually known as 'ok'

// so at a successful type assertion operation, we simply passed the values, to that struct's field 'tracers'
// method that we defined during the type assertion which is 'TraceQueryStart' and 'TraceQueryEnd';
// allowing the tracers field to simply have those methods defined and satisfy them

// Key take away: We're preventing 'fat interfaces' and allows Interface Segragation for this pattern.
// Storage: any(empty intertface) -> Discover: type assertion -> Safety: 'ok' is the guard -> Decoupling ->
// You aren't forcing the objects to conform to a pre-defined set of rules; you are letting them "opt-in" by simply having the right methods.
