package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"ultramedia/worker/internal/jobs"

	jobcontracts "ultramedia/job-contracts"
)

// These tests need a running Redis (make up). They are skipped otherwise.
func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379"
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		t.Skipf("redis not available at %s: %v", url, err)
	}
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

// recorder is a ProcessFunc that behaves according to the message type and
// remembers which job IDs it handled.
type recorder struct {
	mu   sync.Mutex
	jobs []string
}

func (r *recorder) process(_ context.Context, m jobcontracts.Message) error {
	r.mu.Lock()
	r.jobs = append(r.jobs, m.JobID)
	r.mu.Unlock()
	switch m.Type {
	case "ok":
		return nil
	case "fails":
		return errors.New("boom")
	case "panics":
		panic("bad job")
	case "retry":
		return fmt.Errorf("%w: database down", ErrRetry)
	default:
		return fmt.Errorf("%w: %s", jobs.ErrUnsupported, m.Type)
	}
}

func (r *recorder) seen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.jobs)
}

func setup(t *testing.T) (*redis.Client, *recorder, *Consumer) {
	rdb := testRedis(t)
	stream := fmt.Sprintf("test:jobs:%s:%d", t.Name(), time.Now().UnixNano())
	t.Cleanup(func() { rdb.Del(context.Background(), stream) })

	rec := &recorder{}
	cfg := Config{Stream: stream, Group: jobcontracts.ConsumerGroup, Name: "test-worker", Block: 50 * time.Millisecond}
	return rdb, rec, New(rdb, rec.process, slog.New(slog.DiscardHandler), cfg)
}

func add(t *testing.T, rdb *redis.Client, stream string, values map[string]any) {
	t.Helper()
	if err := rdb.XAdd(context.Background(), &redis.XAddArgs{Stream: stream, Values: values}).Err(); err != nil {
		t.Fatal(err)
	}
}

func job(id, jobType string) map[string]any {
	return jobcontracts.Message{JobID: id, Type: jobType, Payload: []byte("{}")}.Values()
}

// runUntil runs the consumer until cond holds, then stops it.
func runUntil(t *testing.T, c *Consumer, cond func() bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("timed out waiting for the consumer")
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("consumer did not stop after cancel")
	}
}

func groupState(t *testing.T, rdb *redis.Client, stream string) (read, pending int64) {
	groups, err := rdb.XInfoGroups(context.Background(), stream).Result()
	if err != nil || len(groups) == 0 {
		return 0, 0
	}
	return groups[0].EntriesRead, groups[0].Pending
}

// deliverWithoutAck simulates a crash: the message reaches this consumer
// but is never acknowledged.
func deliverWithoutAck(t *testing.T, rdb *redis.Client, c *Consumer) {
	t.Helper()
	ctx := context.Background()
	if err := c.EnsureGroup(ctx); err != nil {
		t.Fatal(err)
	}
	err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: c.cfg.Group, Consumer: c.cfg.Name, Streams: []string{c.cfg.Stream, ">"}, Count: 1, Block: -1,
	}).Err()
	if err != nil {
		t.Fatal(err)
	}
}

func TestConsumerAcksEveryMessage(t *testing.T) {
	rdb, rec, c := setup(t)
	stream := c.cfg.Stream
	add(t, rdb, stream, job("j1", "ok"))
	add(t, rdb, stream, job("j2", "fails"))
	add(t, rdb, stream, job("j3", "unknown-type"))
	add(t, rdb, stream, job("j4", "panics"))
	add(t, rdb, stream, map[string]any{"type": "ok"}) // malformed: no job_id/payload
	add(t, rdb, stream, job("j5", "ok"))

	runUntil(t, c, func() bool {
		read, pending := groupState(t, rdb, stream)
		return read == 6 && pending == 0
	})

	if got, want := rec.seen(), []string{"j1", "j2", "j3", "j4", "j5"}; !slices.Equal(got, want) {
		t.Fatalf("processed jobs = %v, want %v", got, want)
	}
}

func TestConsumerRecoversUnackedMessages(t *testing.T) {
	rdb, rec, c := setup(t)
	stream := c.cfg.Stream
	add(t, rdb, stream, job("j1", "ok"))
	deliverWithoutAck(t, rdb, c)
	if _, pending := groupState(t, rdb, stream); pending != 1 {
		t.Fatalf("setup: want 1 pending message, got %d", pending)
	}

	runUntil(t, c, func() bool {
		_, pending := groupState(t, rdb, stream)
		return pending == 0 && len(rec.seen()) == 1
	})
	if got := rec.seen(); got[0] != "j1" {
		t.Fatalf("processed jobs = %v, want [j1]", got)
	}
}

func TestConsumerLeavesRetryMessagesPending(t *testing.T) {
	rdb, rec, c := setup(t)
	stream := c.cfg.Stream
	add(t, rdb, stream, job("j1", "retry"))
	add(t, rdb, stream, job("j2", "ok"))

	runUntil(t, c, func() bool {
		read, _ := groupState(t, rdb, stream)
		return read == 2 && len(rec.seen()) == 2
	})
	if _, pending := groupState(t, rdb, stream); pending != 1 {
		t.Fatalf("want the retry message left pending, got %d pending", pending)
	}

	// On the next start the pending message runs again, exactly once.
	runUntil(t, c, func() bool { return len(rec.seen()) == 3 })
	if got := rec.seen(); got[2] != "j1" {
		t.Fatalf("processed jobs = %v, want j1 again", got)
	}
}
