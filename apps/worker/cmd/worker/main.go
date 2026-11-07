package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"ultramedia/worker/internal/jobs"
	"ultramedia/worker/internal/queue"

	jobcontracts "ultramedia/job-contracts"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return errors.New("REDIS_URL environment variable is required")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("invalid REDIS_URL: %w", err)
	}
	rdb := redis.NewClient(opts)
	defer rdb.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("connect to redis at %s: %w", opts.Addr, err)
	}
	log.Info("connected to redis", "addr", opts.Addr)

	d := jobs.NewDispatcher()
	jobs.RegisterStubs(d, log)

	// A stable name lets a restarted worker pick up jobs it had not finished.
	name := os.Getenv("WORKER_NAME")
	if name == "" {
		name, _ = os.Hostname()
	}

	log.Info("consuming jobs", "stream", jobcontracts.StreamJobs, "group", jobcontracts.ConsumerGroup, "consumer", name)
	if err := queue.New(rdb, d, log, queue.Config{Name: name}).Run(ctx); err != nil {
		return err
	}
	log.Info("worker shut down")
	return nil
}
