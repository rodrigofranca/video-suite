package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"ultramedia/media-core/db"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"
	"ultramedia/worker/internal/jobs"
	"ultramedia/worker/internal/pipeline"
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
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL environment variable is required")
	}
	storageCfg, err := storage.ConfigFromEnv()
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return errors.New("ffmpeg not found in PATH; install FFmpeg to run the worker")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("invalid REDIS_URL: %w", err)
	}
	rdb := redis.NewClient(opts)
	defer rdb.Close()

	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := rdb.Ping(checkCtx).Err(); err != nil {
		return fmt.Errorf("connect to redis at %s: %w", opts.Addr, err)
	}
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()
	files := storage.New(storageCfg)
	if err := files.Check(checkCtx); err != nil {
		return fmt.Errorf("connect to storage at %s: %w", storageCfg.Endpoint, err)
	}
	log.Info("connected", "redis", opts.Addr, "storage", storageCfg.Endpoint)

	st := store.New(pool)
	d := jobs.NewDispatcher()
	jobs.RegisterStubs(d) // extract-metadata and generate-thumbnail are still stubs
	hls := jobs.HLS{Storage: files, Store: st, TempDir: os.Getenv("WORKER_TEMP_DIR")}
	jobs.Handle(d, jobcontracts.JobTypeTranscodeHLS, hls.Transcode)

	runner := &pipeline.Runner{
		Store:      st,
		Dispatcher: d,
		Enqueue: func(ctx context.Context, msg jobcontracts.Message) error {
			return rdb.XAdd(ctx, &redis.XAddArgs{Stream: jobcontracts.StreamJobs, Values: msg.Values()}).Err()
		},
		Log: log,
	}

	// A stable name lets a restarted worker pick up jobs it had not finished.
	name := os.Getenv("WORKER_NAME")
	if name == "" {
		name, _ = os.Hostname()
	}

	log.Info("consuming jobs", "stream", jobcontracts.StreamJobs, "group", jobcontracts.ConsumerGroup, "consumer", name)
	if err := queue.New(rdb, runner.Process, log, queue.Config{Name: name}).Run(ctx); err != nil {
		return err
	}
	log.Info("worker shut down")
	return nil
}
