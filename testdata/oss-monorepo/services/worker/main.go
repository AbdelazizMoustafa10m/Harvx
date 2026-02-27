// Package main runs the Acme background worker service.
//
// The worker dequeues jobs from Redis and processes them concurrently
// using a configurable number of goroutines.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	redisAddr := envOrDefault("REDIS_ADDR", "localhost:6379")
	concurrency := envOrDefaultInt("WORKER_CONCURRENCY", 4)
	queueName := envOrDefault("QUEUE_NAME", "acme:jobs")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	slog.Info("starting worker",
		"redis_addr", redisAddr,
		"concurrency", concurrency,
		"queue", queueName,
	)

	h := NewHandler(HandlerConfig{
		RedisAddr:   redisAddr,
		QueueName:   queueName,
		Concurrency: concurrency,
	})

	if err := h.Run(ctx); err != nil {
		slog.Error("worker exited with error", "error", err)
		os.Exit(1)
	}

	slog.Info("worker shut down gracefully")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}