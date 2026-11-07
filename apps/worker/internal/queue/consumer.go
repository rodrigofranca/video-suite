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

// Config controls which stream and consumer group the worker reads from.
type Config struct {
	Stream string        // defaults to jobcontracts.StreamJobs
	Group  string        // defaults to jobcontracts.ConsumerGroup
	Name   string        // consumer name; keep it stable across restarts
	Block  time.Duration // how long one read waits for new messages; defaults to 2s
}

// Consumer reads one message at a time and dispatches it by job type.
type Consumer struct {
	rdb        *redis.Client
	dispatcher *jobs.Dispatcher
	log        *slog.Logger
	cfg        Config
}

func New(rdb *redis.Client, d *jobs.Dispatcher, log *slog.Logger, cfg Config) *Consumer {
	if cfg.Stream == "" {
		cfg.Stream = jobcontracts.StreamJobs
	}
	if cfg.Group == "" {
		cfg.Group = jobcontracts.ConsumerGroup
	}
	if cfg.Block <= 0 {
		cfg.Block = 2 * time.Second
	}
	return &Consumer{rdb: rdb, dispatcher: d, log: log, cfg: cfg}
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
	for ctx.Err() == nil {
		n, err := c.readOnce(ctx, "0")
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("recover pending jobs: %w", err)
		}
		if n == 0 {
			return nil
		}
	}
	return nil
}

// readOnce reads and processes at most one message. id is ">" for new
// messages or "0" for messages already delivered to this consumer.
func (c *Consumer) readOnce(ctx context.Context, id string) (int, error) {
	args := &redis.XReadGroupArgs{
		Group:    c.cfg.Group,
		Consumer: c.cfg.Name,
		Streams:  []string{c.cfg.Stream, id},
		Count:    1,
		Block:    -1, // history reads return immediately
	}
	if id == ">" {
		args.Block = c.cfg.Block
	}
	streams, err := c.rdb.XReadGroup(ctx, args).Result()
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
			if err := c.process(ctx, msg); err != nil {
				return n, err
			}
		}
	}
	return n, nil
}

// process dispatches one message and always acknowledges it, so a failing or
// unknown job never blocks the stream. Job status lives in PostgreSQL.
func (c *Consumer) process(ctx context.Context, xmsg redis.XMessage) error {
	log := c.log.With("message_id", xmsg.ID)
	msg, err := jobcontracts.ParseMessage(xmsg.Values)
	if err != nil {
		log.Error("invalid job message", "error", err)
	} else {
		log = log.With("job_id", msg.JobID, "type", msg.Type)
		switch err := c.dispatch(ctx, msg); {
		case errors.Is(err, jobs.ErrUnsupported):
			log.Warn("unsupported job type: " + msg.Type)
		case err != nil:
			log.Error("job failed", "error", err)
		default:
			log.Info("job finished")
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

// dispatch runs the handler and turns a panic into an error, so one bad job
// cannot crash the worker in a loop.
func (c *Consumer) dispatch(ctx context.Context, msg jobcontracts.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return c.dispatcher.Dispatch(ctx, msg)
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
