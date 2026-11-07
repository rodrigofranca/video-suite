// Package queue consumes jobs from a Redis stream using a consumer group.
package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"ultramedia/worker/internal/jobs"

	jobcontracts "ultramedia/job-contracts"
)

// ErrRetry tells the consumer not to ack a message, so it runs again when
// the worker restarts (for example when the job was interrupted by shutdown
// or the database was unreachable).
var ErrRetry = errors.New("will retry after restart")

// ProcessFunc handles one message.
type ProcessFunc func(ctx context.Context, msg jobcontracts.Message) error

// Config controls which stream and consumer group the worker reads from.
type Config struct {
	Stream string        // defaults to jobcontracts.StreamJobs
	Group  string        // defaults to jobcontracts.ConsumerGroup
	Name   string        // consumer name; keep it stable across restarts
	Block  time.Duration // how long one read waits for new messages; defaults to 2s
}

// Consumer reads one message at a time and hands it to a ProcessFunc.
type Consumer struct {
	rdb     *redis.Client
	process ProcessFunc
	log     *slog.Logger
	cfg     Config
}

func New(rdb *redis.Client, process ProcessFunc, log *slog.Logger, cfg Config) *Consumer {
	if cfg.Stream == "" {
		cfg.Stream = jobcontracts.StreamJobs
	}
	if cfg.Group == "" {
		cfg.Group = jobcontracts.ConsumerGroup
	}
	if cfg.Block <= 0 {
		cfg.Block = 2 * time.Second
	}
	return &Consumer{rdb: rdb, process: process, log: log, cfg: cfg}
}

// EnsureGroup creates the stream and consumer group if they do not exist.
func (c *Consumer) EnsureGroup(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, c.cfg.Stream, c.cfg.Group, "0").Err()
	if err != nil && !strings.HasPrefix(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create consumer group: %w", err)
	}
	return nil
}

// Run consumes jobs until ctx is canceled. It first re-runs messages that were
// delivered to this consumer but never acknowledged (for example after a
// crash), then waits for new ones.
func (c *Consumer) Run(ctx context.Context) error {
	if err := c.EnsureGroup(ctx); err != nil {
		return err
	}
	if err := c.recoverPending(ctx); err != nil {
		return err
	}
	for ctx.Err() == nil {
		if _, err := c.readOnce(ctx, ">"); err != nil && ctx.Err() == nil {
			c.log.Error("read from stream failed, retrying", "error", err)
			sleep(ctx, time.Second)
		}
	}
	return nil
}

func (c *Consumer) recoverPending(ctx context.Context) error {
	// Read the history page by page: a message left unacked on purpose
	// (ErrRetry) must not be read again in a loop.
	after := "0"
	for ctx.Err() == nil {
		streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group: c.cfg.Group, Consumer: c.cfg.Name, Streams: []string{c.cfg.Stream, after}, Count: 1, Block: -1,
		}).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("recover pending jobs: %w", err)
		}
		if len(streams) == 0 || len(streams[0].Messages) == 0 {
			return nil
		}
		msg := streams[0].Messages[0]
		after = msg.ID
		c.log.Info("resuming unfinished job", "message_id", msg.ID)
		if err := c.handle(ctx, msg); err != nil {
			return fmt.Errorf("recover pending jobs: %w", err)
		}
	}
	return nil
}

// readOnce waits for one new message and processes it.
func (c *Consumer) readOnce(ctx context.Context, id string) (int, error) {
	streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: c.cfg.Group, Consumer: c.cfg.Name, Streams: []string{c.cfg.Stream, id}, Count: 1, Block: c.cfg.Block,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil // no new messages before the block timeout
	}
	if err != nil {
		return 0, err
	}
	n := 0
	for _, s := range streams {
		for _, msg := range s.Messages {
			n++
			if err := c.handle(ctx, msg); err != nil {
				return n, err
			}
		}
	}
	return n, nil
}

// handle processes one message and acknowledges it, so a failing or unknown
// job never blocks the stream (job status lives in PostgreSQL). The only
// exception is ErrRetry, which leaves the message for the next start.
func (c *Consumer) handle(ctx context.Context, xmsg redis.XMessage) error {
	log := c.log.With("message_id", xmsg.ID)
	msg, err := jobcontracts.ParseMessage(xmsg.Values)
	if err != nil {
		log.Error("invalid job message", "error", err)
	} else {
		log = log.With("job_id", msg.JobID, "type", msg.Type)
		switch err := c.run(ctx, msg); {
		case errors.Is(err, ErrRetry):
			log.Warn("job not acknowledged; it will run again after restart", "reason", err)
			return nil
		case errors.Is(err, jobs.ErrUnsupported):
			log.Warn("unsupported job type: " + msg.Type)
		case err != nil:
			log.Error("job failed", "error", err)
		}
	}

	// Ack even if ctx was canceled mid-job, so finished work is not redone.
	ackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := c.rdb.XAck(ackCtx, c.cfg.Stream, c.cfg.Group, xmsg.ID).Err(); err != nil {
		return fmt.Errorf("ack message %s: %w", xmsg.ID, err)
	}
	return nil
}

// run calls the ProcessFunc and turns a panic into an error, so one bad job
// cannot crash the worker in a loop.
func (c *Consumer) run(ctx context.Context, msg jobcontracts.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return c.process(ctx, msg)
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
