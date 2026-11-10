package outbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ultramedia/media-core/db"
	"ultramedia/media-core/domain"
	"ultramedia/media-core/store"

	jobcontracts "ultramedia/job-contracts"
)

// These tests use the migrated database from `make up && make migrate` and
// are skipped without it. Each relay only touches its own test's video.

type harness struct {
	t     *testing.T
	pool  *pgxpool.Pool
	store *store.Store
	// The video under test and its jobs, and every video the relays may touch.
	asset  string
	jobs   []domain.Job
	assets []string

	mu   sync.Mutex
	sent []jobcontracts.Message
	down error // what the fake Redis answers, nil when it works
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://ultramedia:ultramedia@localhost:5432/ultramedia?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	if _, err := pool.Exec(ctx, "SELECT 1 FROM outbox_events LIMIT 1"); err != nil {
		pool.Close()
		t.Skipf("database not migrated: %v", err)
	}
	t.Cleanup(pool.Close)
	h := &harness{t: t, pool: pool, store: store.New(pool)}
	h.asset, h.jobs = h.upload()
	return h
}

// upload creates a video with the whole pipeline, as the API does.
func (h *harness) upload() (string, []domain.Job) {
	ctx := context.Background()
	asset := domain.Asset{ID: domain.NewID(), Filename: "talk.mp4", Status: domain.AssetUploading}
	if err := h.store.CreateAsset(ctx, asset); err != nil {
		h.t.Fatal(err)
	}
	h.t.Cleanup(func() { h.store.DeleteAsset(context.Background(), asset.ID) })
	base := time.Now()
	var jobs []domain.Job
	for i, typ := range jobcontracts.Pipeline {
		jobs = append(jobs, domain.Job{ID: domain.NewID(), AssetID: asset.ID, Type: typ, InputJSON: fmt.Appendf(nil, `{"step":%d}`, i),
			Status: domain.JobPending, CreatedAt: base.Add(time.Duration(i) * time.Millisecond)})
	}
	src := domain.Source{ID: domain.NewID(), AssetID: asset.ID, StorageKey: "k", MimeType: "video/mp4", SizeBytes: 1, Checksum: "c"}
	if err := h.store.CompleteUpload(ctx, src, jobs); err != nil {
		h.t.Fatal(err)
	}
	h.assets = append(h.assets, asset.ID)
	return asset.ID, jobs
}

func (h *harness) relay() *Relay {
	return &Relay{Store: h.store, Handlers: Handlers(h.send), Assets: h.assets}
}

func (h *harness) send(_ context.Context, msg jobcontracts.Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.down != nil {
		return h.down
	}
	h.sent = append(h.sent, msg)
	return nil
}

func (h *harness) sentIDs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var ids []string
	for _, m := range h.sent {
		ids = append(ids, m.JobID)
	}
	return ids
}

func (h *harness) events() []domain.Event {
	h.t.Helper()
	events, err := h.store.Events(context.Background(), h.asset)
	if err != nil {
		h.t.Fatal(err)
	}
	return events
}

func (h *harness) status() string {
	a, err := h.store.GetAsset(context.Background(), h.asset)
	if err != nil {
		h.t.Fatal(err)
	}
	return a.Status
}

// due makes every waiting event of the video due now, as if time passed.
func (h *harness) due() {
	if _, err := h.pool.Exec(context.Background(), `UPDATE outbox_events SET next_attempt_at = NOW() WHERE asset_id = $1`, h.asset); err != nil {
		h.t.Fatal(err)
	}
}

func TestUploadQueuesOnlyTheFirstJob(t *testing.T) {
	h := newHarness(t)
	events := h.events()
	if len(events) != 1 || events[0].Type != jobcontracts.EventJobQueued || events[0].JobID != h.jobs[0].ID || events[0].HandledAt != nil {
		t.Fatalf("events = %+v", events)
	}
	if n := h.relay().Drain(context.Background()); n != 1 {
		t.Fatalf("handled %d events", n)
	}
	if ids := h.sentIDs(); len(ids) != 1 || ids[0] != h.jobs[0].ID || h.sent[0].Type != jobcontracts.JobTypeExtractMetadata ||
		!strings.Contains(string(h.sent[0].Payload), `"step": 0`) {
		t.Fatalf("sent %+v", h.sent)
	}
	if e := h.events()[0]; e.HandledAt == nil || e.Attempts != 1 {
		t.Fatalf("event after the relay: %+v", e)
	}
	if h.relay().Drain(context.Background()) != 0 || len(h.sentIDs()) != 1 {
		t.Fatal("a handled event was handled again")
	}
}

func TestFinishedJobsMoveThePipelineOn(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	r := h.relay()
	r.Drain(ctx)
	for i, job := range h.jobs {
		var err error
		if job.Type == jobcontracts.JobTypeExtractMetadata {
			err = errors.New("ffprobe failed") // a failure does not stop the pipeline
		}
		h.store.StartJob(ctx, job.ID)
		if err := h.store.FinishJob(ctx, job.ID, err); err != nil {
			t.Fatal(err)
		}
		r.Drain(ctx)
		ids := h.sentIDs()
		if i+1 < len(h.jobs) && (len(ids) != i+2 || ids[i+1] != h.jobs[i+1].ID) {
			t.Fatalf("after %s: sent %v", job.Type, ids)
		}
		if want := domain.AssetProcessing; job.Type == jobcontracts.JobTypeExtractMetadata && h.status() != want {
			t.Fatalf("after a failed %s the asset is %s", job.Type, h.status())
		}
		if job.Type == jobcontracts.JobTypeTranscodeHLS && h.status() != domain.AssetReady {
			t.Fatalf("after HLS the asset is %s, want ready", h.status())
		}
	}
	if ids := h.sentIDs(); len(ids) != len(h.jobs) {
		t.Fatalf("sent %d messages for %d jobs", len(ids), len(h.jobs))
	}
	for _, e := range h.events() {
		if e.HandledAt == nil {
			t.Fatalf("event left: %+v", e)
		}
	}
}

func TestFailedHLSFailsTheAsset(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	hls := h.jobs[2]
	h.store.FinishJob(ctx, hls.ID, errors.New("encoder crashed"))
	h.relay().Drain(ctx)
	if h.status() != domain.AssetFailed {
		t.Fatalf("asset %s, want failed", h.status())
	}
}

func TestEventsWaitWhileRedisIsDown(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.down = errors.New("connection refused")
	if n := h.relay().Drain(ctx); n != 0 {
		t.Fatalf("handled %d events with Redis down", n)
	}
	e := h.events()[0]
	if e.HandledAt != nil || e.Attempts != 1 || e.LastError == nil {
		t.Fatalf("event after a failed send: %+v", e)
	}
	job, _ := h.store.GetJob(ctx, h.jobs[0].ID)
	if job.Status != domain.JobPending {
		t.Fatalf("the job is %s; it should wait, pending", job.Status)
	}
	// It is not tried again before its time...
	h.down = nil
	if h.relay().Drain(ctx) != 0 {
		t.Fatal("tried again too soon")
	}
	// ...and once it is due, it goes.
	h.due()
	if h.relay().Drain(ctx) != 1 || len(h.sentIDs()) != 1 {
		t.Fatalf("sent %v once Redis was back", h.sentIDs())
	}
}

func TestAJobThatAlreadyRanIsNotSentAgain(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	// The relay stopped after sending, before marking: the job ran meanwhile.
	h.store.StartJob(ctx, h.jobs[0].ID)
	h.relay().Drain(ctx)
	if ids := h.sentIDs(); len(ids) != 0 {
		t.Fatalf("a running job was sent again: %v", ids)
	}
}

func TestRelaysShareTheWork(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	want := map[string]bool{h.jobs[0].ID: true}
	for range 5 {
		_, jobs := h.upload()
		want[jobs[0].ID] = true
	}
	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() { h.relay().Drain(ctx) })
	}
	wg.Wait()
	seen := map[string]int{}
	for _, id := range h.sentIDs() {
		seen[id]++
	}
	if len(seen) != len(want) {
		t.Fatalf("sent %d distinct jobs, want %d", len(seen), len(want))
	}
	for id, n := range seen {
		if !want[id] || n != 1 {
			t.Fatalf("job %s sent %d times", id, n)
		}
	}
}

func TestUnknownEventsAreSetAside(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	if _, err := h.pool.Exec(ctx, `INSERT INTO outbox_events (type, asset_id) VALUES ('video.dreamed', $1)`, h.asset); err != nil {
		t.Fatal(err)
	}
	if n := h.relay().Drain(ctx); n != 2 {
		t.Fatalf("handled %d events, want the queued job and the unknown one", n)
	}
}

func TestListenerWakesOnNewEvents(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	l, err := h.store.Listen(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		time.Sleep(50 * time.Millisecond)
		h.store.FinishJob(ctx, h.jobs[0].ID, nil)
	}()
	start := time.Now()
	if err := l.Wait(ctx, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if waited := time.Since(start); waited > 2*time.Second {
		t.Fatalf("woke after %s; the notification did not arrive", waited)
	}
	// A timeout is not an error: the relay looks anyway. (Other tests'
	// events may wake it sooner, so the time is not checked.)
	if err := l.Wait(ctx, 200*time.Millisecond); err != nil {
		t.Fatalf("idle wait: %v", err)
	}
}

func TestHandledEventsArePruned(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.relay().Drain(ctx)
	if _, err := h.pool.Exec(ctx, `UPDATE outbox_events SET handled_at = NOW() - INTERVAL '8 days' WHERE asset_id = $1`, h.asset); err != nil {
		t.Fatal(err)
	}
	if _, err := h.store.PruneEvents(ctx, DefaultKeepFor); err != nil {
		t.Fatal(err)
	}
	if events := h.events(); len(events) != 0 {
		t.Fatalf("events left: %+v", events)
	}
}

func TestRetryDelayGrows(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.down = errors.New("down")
	var waits []time.Duration
	for range 3 {
		h.relay().Drain(ctx)
		var wait time.Duration
		h.pool.QueryRow(ctx, `SELECT EXTRACT(EPOCH FROM next_attempt_at - NOW()) * 1000000000 FROM outbox_events WHERE asset_id = $1`, h.asset).Scan(&wait)
		waits = append(waits, wait)
		h.due()
	}
	if !(waits[0] < waits[1] && waits[1] < waits[2] && waits[2] <= 4*time.Second+time.Second) {
		t.Fatalf("waits %v, want about 1 s, 2 s, 4 s", waits)
	}
}
