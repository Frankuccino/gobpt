package job

import (
	"github.com/Frankuccino/gobpt/internal/config"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

// This JobService struct includes methods like:
// Start() - for starting the task handlers with handleWelcomeEmailTask as the handler
// Stop() - for logging & gracefully shutting down the active workers and closing the redis connection
// InitHandlers() - for initializing an email client with Integration config API Key
// handleWelcomeEmailTask() - for processing a background welcome email dispatch
type JobService struct {
	Client *asynq.Client
	server *asynq.Server
	logger *zerolog.Logger
}

func NewJobService(logger *zerolog.Logger, cfg *config.Config) *JobService {
	redisAddr := cfg.Redis.Address

	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr: redisAddr,
	})

	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6, // Higher priority queue for important emails
				"default":  3, // Default priority for most emails
				"low":      1, // Lower priority for non-urgen emails
			},
		},
	)

	return &JobService{
		Client: client,
		server: server,
		logger: logger,
	}
}

func (j *JobService) Start() error {
	// Register task handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskWelcome, j.handleWelcomeEmailTask)

	j.logger.Info().Msg("Starting background job server")
	if err := j.server.Start(mux); err != nil {
		return err
	}
	return nil
}

func (j *JobService) Stop() {
	j.logger.Info().Msg("Stopping background job server")
	j.server.Shutdown()
	j.Client.Close()
}

// We'll use a library called Asynq which uses redis behind the scenes to process background jobs,
// execute them, resend, to reexcute, exponential backoff and a lot of features.
// We'll just keep it simple, initialize a server and use that server to perform whatever
// background jobs we have.

// We're following here the pattern of dependency injection which basically means;
// Instead of importing global variable, instead we want to take the logger as a dependency
// through our parameter and that's how we want to use it.
// One benefit to that is when we want to test our application, or different modules, either unit test
// or integration test. Since we are not using any global variables, we can specifically control what
// kind of logger that we use in our test environments.
// Second, it makes your application more explicity and reduced bugs. It's a good pattern and follow it
// as much as we can.
