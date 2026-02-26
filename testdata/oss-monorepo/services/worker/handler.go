package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Job represents a background job to be processed.
type Job struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Attempts   int             `json:"attempts"`
	MaxRetries int             `json:"max_retries"`
	CreatedAt  time.Time       `json:"created_at"`
}

// HandlerConfig configures the worker handler.
type HandlerConfig struct {
	RedisAddr   string
	QueueName   string
	Concurrency int
}

// Handler processes background jobs from a queue.
type Handler struct {
	config HandlerConfig
}

// NewHandler creates a new job handler with the given configuration.
func NewHandler(cfg HandlerConfig) *Handler {
	return &Handler{config: cfg}
}

// Run starts the worker loop, processing jobs until the context is cancelled.
func (h *Handler) Run(ctx context.Context) error {
	slog.Info("worker handler starting",
		"concurrency", h.config.Concurrency,
		"queue", h.config.QueueName,
	)

	var wg sync.WaitGroup
	sem := make(chan struct{}, h.config.Concurrency)

	for {
		select {
		case <-ctx.Done():
			slog.Info("context cancelled, draining workers...")
			wg.Wait()
			return nil
		default:
		}

		job, err := h.dequeue(ctx)
		if err != nil {
			slog.Warn("dequeue error", "error", err)
			time.Sleep(time.Second)
			continue
		}

		if job == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(j *Job) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := h.processJob(ctx, j); err != nil {
				slog.Error("job failed",
					"job_id", j.ID,
					"type", j.Type,
					"attempt", j.Attempts,
					"error", err,
				)
				return
			}

			slog.Info("job completed",
				"job_id", j.ID,
				"type", j.Type,
			)
		}(job)
	}
}

func (h *Handler) dequeue(_ context.Context) (*Job, error) {
	// Placeholder: in production, this would use Redis BRPOP
	return nil, nil
}

func (h *Handler) processJob(ctx context.Context, job *Job) error {
	slog.Info("processing job",
		"job_id", job.ID,
		"type", job.Type,
	)

	switch job.Type {
	case "email.send":
		return h.handleEmailSend(ctx, job)
	case "report.generate":
		return h.handleReportGenerate(ctx, job)
	case "data.export":
		return h.handleDataExport(ctx, job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

func (h *Handler) handleEmailSend(_ context.Context, job *Job) error {
	slog.Debug("sending email", "job_id", job.ID)
	// Simulate work
	time.Sleep(50 * time.Millisecond)
	return nil
}

func (h *Handler) handleReportGenerate(_ context.Context, job *Job) error {
	slog.Debug("generating report", "job_id", job.ID)
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (h *Handler) handleDataExport(_ context.Context, job *Job) error {
	slog.Debug("exporting data", "job_id", job.ID)
	time.Sleep(100 * time.Millisecond)
	return nil
}