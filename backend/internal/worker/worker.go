package worker

import (
	"context"
	"log/slog"
	"time"
)

type Runner struct {
	logger       *slog.Logger
	pollInterval time.Duration
	workerID     string
	processor    Processor
}

type Processor interface {
	ProcessNext(context.Context, string) (bool, error)
}

func New(logger *slog.Logger, pollInterval time.Duration, workerID string, processor Processor) *Runner {
	return &Runner{
		logger: logger, pollInterval: pollInterval, workerID: workerID, processor: processor,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	r.logger.Info("worker_started", "poll_interval", r.pollInterval.String())
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("worker_stopped")
			return nil
		case <-timer.C:
			processed, err := r.processor.ProcessNext(ctx, r.workerID)
			if err != nil {
				r.logger.Error("worker_job_failed", "error", err)
			}
			delay := r.pollInterval
			if processed && err == nil {
				delay = 10 * time.Millisecond
			}
			timer.Reset(delay)
		}
	}
}
