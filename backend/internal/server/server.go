package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Frankuccino/gobpt/internal/config"
	"github.com/Frankuccino/gobpt/internal/database"
	"github.com/Frankuccino/gobpt/internal/lib/job"
	"github.com/Frankuccino/gobpt/internal/logger"
	"github.com/newrelic/go-agent/v3/integrations/nrredis-v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Server struct {
	Config        *config.Config
	Logger        *zerolog.Logger
	LoggerService *logger.LoggerService
	DB            *database.Database
	Redis         *redis.Client
	httpServer    *http.Server
	Job           *job.JobService
}

func New(cfg *config.Config, logger *zerolog.Logger, loggerService *logger.LoggerService) (*Server, error) {
	db, err := database.New(cfg, logger, loggerService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Redis client with New Relic integration
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Address,
	})

	// Add New Relic Redis hooks if available
	if loggerService != nil && loggerService.GetApplication() != nil {
		redisClient.AddHook(nrredis.NewHook(redisClient.Options()))
		// We can pass this observability tracer inside our redis client so that we get all logs in new relic
	}

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to connect to Redis, continuing without Redis")
		// Don't fail startup if Redis is unavailable
	}

	// job service
	jobService := job.NewJobService(logger, cfg)
	jobService.InitHandlers(cfg, logger)

	// Start job server
	if err := jobService.Start(); err != nil {
		return nil, err
	}

	server := &Server{
		Config:        cfg,
		Logger:        logger,
		LoggerService: loggerService,
		DB:            db,
		Redis:         redisClient,
		Job:           jobService,
	}

	// Start metrics collection
	// Runtime metrics are automatically collected by New Relic Go agent

	return server, nil
}

// this will contain the core data structure which will contain the:
// config,
// logger,
// New Relic service,
// database instance,
// redis instance,
// HTTP server, and
// background job processing server.
// we'll take all these different instances of different servers/services and modules and
// put them inside a single data structure, and we call that as a server.
// Using that server, by passing an instance, a pointer to the server, we can establish this
// dependency injection workflow. So that we'll pass the pointers everywhere when we need on of these things.

// We'll then implement a function which would initialize all these different services, servers, and
// it is going to put all of them, the initialized versions inside this struct.
