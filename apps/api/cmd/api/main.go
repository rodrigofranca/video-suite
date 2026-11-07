package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"ultramedia/api/internal/httpapi"
	"ultramedia/media-core/db"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"

	jobcontracts "ultramedia/job-contracts"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL environment variable is required")
	}
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return errors.New("REDIS_URL environment variable is required")
	}
	storageCfg, err := storage.ConfigFromEnv()
	if err != nil {
		return err
	}
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	origin := os.Getenv("DASHBOARD_ORIGIN")
	if origin == "" {
		origin = "http://localhost:5173"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("invalid REDIS_URL: %w", err)
	}
	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()

	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := rdb.Ping(checkCtx).Err(); err != nil {
		return fmt.Errorf("connect to redis at %s: %w", redisOpts.Addr, err)
	}
	files := storage.New(storageCfg)
	if err := files.Check(checkCtx); err != nil {
		return fmt.Errorf("connect to storage at %s: %w", storageCfg.Endpoint, err)
	}

	srv := &http.Server{
		Addr: ":" + port,
		Handler: httpapi.NewRouter(httpapi.Deps{
			DB:      pool,
			Store:   store.New(pool),
			Storage: files,
			Enqueue: func(ctx context.Context, msg jobcontracts.Message) error {
				return rdb.XAdd(ctx, &redis.XAddArgs{Stream: msg.Stream(), Values: msg.Values()}).Err()
			},
			AllowedOrigin: origin,
			Log:           log,
		}),
		// Only the header read is time-limited here: uploads of up to 2GB need
		// long request bodies, so the upload handler enforces its own limits.
		ReadHeaderTimeout: 10 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		log.Info("API listening", "addr", srv.Addr, "dashboard_origin", origin)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
