package pipeline

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"ultramedia/media-core/db"
	"ultramedia/media-core/domain"
	"ultramedia/media-core/outbox"
	"ultramedia/media-core/store"
	"ultramedia/worker/internal/jobs"
	"ultramedia/worker/internal/queue"

	jobcontracts "ultramedia/job-contracts"
)

// These tests use the migrated database from `make up && make migrate`.

type harness struct {
	t      *testing.T
	store  *store.Store
	runner *Runner
	// The outbox relay, for this test's video only: it moves the pipeline on
	// after each job, as the one in the worker process does.
	relay *outbox.Relay
	asset domain.Asset
	jobs  []domain.Job // pipeline order

	mu        sync.Mutex
	calls     map[string]int // handler calls per job type
	enqueued  []jobcontracts.Message
	behavior  map[string]func(ctx context.Context) error
	redisDown bool
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
	t.Cleanup(pool.Close)

	h := &harness{t: t, store: store.New(pool), calls: map[string]int{}, behavior: map[string]func(context.Context) error{}}
	h.asset = domain.Asset{ID: domain.NewID(), Filename: "clip.mp4", Status: domain.AssetUploading}
	if err := h.store.CreateAsset(ctx, h.asset); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.store.DeleteAsset(context.Background(), h.asset.ID) })
	base := time.Now()
	for i, typ := range jobcontracts.Pipeline {
		h.jobs = append(h.jobs, domain.Job{ID: domain.NewID(), AssetID: h.asset.ID, Type: typ, InputJSON: []byte(`{}`), Status: domain.JobPending, CreatedAt: base.Add(time.Duration(i) * time.Millisecond)})
	}
	src := domain.Source{ID: domain.NewID(), AssetID: h.asset.ID, StorageKey: "k", MimeType: "video/mp4", SizeBytes: 1, Checksum: "x"}
	if err := h.store.CompleteUpload(ctx, src, h.jobs); err != nil {
		t.Fatal(err)
	}

	d := jobs.NewDispatcher()
	for _, typ := range jobcontracts.Pipeline {
		d.Register(typ, func(ctx context.Context, job jobs.Job) error {
			h.mu.Lock()
			h.calls[typ]++
			fn := h.behavior[typ]
			h.mu.Unlock()
			job.Log.Info("working")
			if fn != nil {
				return fn(ctx)
			}
			return nil
		})
	}
	h.runner = &Runner{Store: h.store, Dispatcher: d, Timeout: 5 * time.Second}
	h.relay = &outbox.Relay{Store: h.store, Assets: []string{h.asset.ID}, Handlers: outbox.Handlers(func(_ context.Context, m jobcontracts.Message) error {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.redisDown {
			return errors.New("redis down")
		}
		h.enqueued = append(h.enqueued, m)
		return nil
	})}
	h.relay.Drain(ctx) // the upload queued the first job
	h.enqueued = nil
	return h
}

// process runs a job, then lets the relay handle what it recorded.
func (h *harness) process(ctx context.Context, i int) error {
	j := h.jobs[i]
	err := h.runner.Process(ctx, jobcontracts.Message{JobID: j.ID, Type: j.Type, Payload: j.InputJSON})
	h.relay.Drain(context.Background())
	return err
}

func (h *harness) job(i int) domain.Job {
	j, err := h.store.GetJob(context.Background(), h.jobs[i].ID)
	if err != nil {
		h.t.Fatal(err)
	}
	return j
}

func (h *harness) assetStatus() string {
	a, _ := h.store.GetAsset(context.Background(), h.asset.ID)
	return a.Status
}

func (h *harness) lastEnqueued() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.enqueued) == 0 {
		return ""
	}
	return h.enqueued[len(h.enqueued)-1].JobID
}

func TestPipelineHappyPath(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	const hls = 2 // index of transcode-hls in the pipeline
	last := len(h.jobs) - 1
	for i := range h.jobs {
		if err := h.process(ctx, i); err != nil {
			t.Fatalf("job %d: %v", i, err)
		}
		if got := h.job(i); got.Status != domain.JobDone || got.Progress != 100 {
			t.Fatalf("job %d = %s/%d, want done/100", i, got.Status, got.Progress)
		}
		if i < last && h.lastEnqueued() != h.jobs[i+1].ID {
			t.Fatalf("after job %d the next job was not enqueued", i)
		}
		want := domain.AssetProcessing
		if i >= hls {
			want = domain.AssetReady // playable while audio and transcription run
		}
		if got := h.assetStatus(); got != want {
			t.Fatalf("after job %d (%s) asset = %s, want %s", i, h.jobs[i].Type, got, want)
		}
	}
	if len(h.enqueued) != last {
		t.Fatalf("enqueued %d messages, want %d", len(h.enqueued), last)
	}
	// Each job goes to the stream of the worker that runs it.
	for _, m := range h.enqueued {
		want := jobcontracts.StreamJobs
		switch m.Type {
		case jobcontracts.JobTypeTranscribeAudio, jobcontracts.JobTypeAnalyzeTranscript, jobcontracts.JobTypeSuggestClips, jobcontracts.JobTypeTrackFaces:
			want = jobcontracts.StreamAIJobs
		}
		if m.Stream() != want {
			t.Fatalf("%s goes to %s, want %s", m.Type, m.Stream(), want)
		}
	}

	logs, _ := h.store.ListJobLogs(ctx, h.jobs[0].ID)
	var msgs []string
	for _, l := range logs {
		msgs = append(msgs, l.Message)
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{"job started", "working", "job finished", "queued next job next_type=generate-thumbnail"} {
		if !strings.Contains(all, want) {
			t.Errorf("job logs missing %q:\n%s", want, all)
		}
	}
}

func TestMetadataFailureDoesNotStopPipeline(t *testing.T) {
	h := newHarness(t)
	h.behavior[jobcontracts.JobTypeExtractMetadata] = func(context.Context) error { return errors.New("ffprobe: invalid data") }

	if err := h.process(context.Background(), 0); err == nil {
		t.Fatal("want the handler error")
	}
	got := h.job(0)
	if got.Status != domain.JobFailed || got.ErrorMessage == nil || *got.ErrorMessage != "ffprobe: invalid data" {
		t.Fatalf("job = %+v", got)
	}
	if h.lastEnqueued() != h.jobs[1].ID || h.assetStatus() != domain.AssetProcessing {
		t.Fatalf("pipeline should continue: enqueued %q, asset %s", h.lastEnqueued(), h.assetStatus())
	}
	logs, _ := h.store.ListJobLogs(context.Background(), h.jobs[0].ID)
	if last := logs[len(logs)-2]; last.Level != domain.LogError || !strings.Contains(last.Message, "ffprobe: invalid data") {
		t.Fatalf("missing error log, got %+v", logs)
	}
}

func TestHLSFailureFailsAsset(t *testing.T) {
	h := newHarness(t)
	h.behavior[jobcontracts.JobTypeTranscodeHLS] = func(context.Context) error { return errors.New("ffmpeg failed") }
	h.process(context.Background(), 2)
	if h.assetStatus() != domain.AssetFailed {
		t.Fatalf("asset = %s, want failed", h.assetStatus())
	}
}

func TestDuplicateDeliveryIsSkipped(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.process(ctx, 0)
	if err := h.process(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if h.calls[jobcontracts.JobTypeExtractMetadata] != 1 {
		t.Fatalf("handler ran %d times, want 1", h.calls[jobcontracts.JobTypeExtractMetadata])
	}
}

func TestTimeoutAndPanicFailTheJob(t *testing.T) {
	h := newHarness(t)
	h.runner.Timeout = 50 * time.Millisecond
	h.behavior[jobcontracts.JobTypeExtractMetadata] = func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
	h.behavior[jobcontracts.JobTypeGenerateThumbnail] = func(context.Context) error { panic("nil map") }

	h.process(context.Background(), 0)
	if got := h.job(0); got.Status != domain.JobFailed || !strings.Contains(*got.ErrorMessage, "timed out after 50ms") {
		t.Fatalf("timed-out job = %s %v", got.Status, *got.ErrorMessage)
	}
	h.process(context.Background(), 1)
	if got := h.job(1); got.Status != domain.JobFailed || !strings.Contains(*got.ErrorMessage, "handler panic: nil map") {
		t.Fatalf("panicked job = %s %v", got.Status, *got.ErrorMessage)
	}
}

func TestShutdownLeavesJobToResume(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	h.behavior[jobcontracts.JobTypeExtractMetadata] = func(runCtx context.Context) error {
		cancel() // the worker receives SIGTERM mid-job
		<-runCtx.Done()
		return runCtx.Err()
	}

	err := h.process(ctx, 0)
	if !errors.Is(err, queue.ErrRetry) {
		t.Fatalf("want ErrRetry so the message is not acked, got %v", err)
	}
	if got := h.job(0); got.Status != domain.JobRunning {
		t.Fatalf("job = %s, want it left running", got.Status)
	}
	if h.lastEnqueued() != "" {
		t.Fatal("no next job should be enqueued")
	}

	// On restart the same message runs again and completes.
	delete(h.behavior, jobcontracts.JobTypeExtractMetadata)
	if err := h.process(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if got := h.job(0); got.Status != domain.JobDone {
		t.Fatalf("job = %s after restart, want done", got.Status)
	}
}

func TestNextJobWaitsWhileRedisIsDown(t *testing.T) {
	h := newHarness(t)
	h.redisDown = true
	h.process(context.Background(), 0)
	// The next job is queued in the outbox, not failed: it goes once Redis is back.
	if got := h.job(1); got.Status != domain.JobPending || h.assetStatus() != domain.AssetProcessing || h.lastEnqueued() != "" {
		t.Fatalf("next job %s, asset %s, sent %q", got.Status, h.assetStatus(), h.lastEnqueued())
	}
	events, _ := h.store.Events(context.Background(), h.asset.ID)
	last := events[len(events)-1]
	if last.Type != jobcontracts.EventJobQueued || last.JobID != h.jobs[1].ID || last.HandledAt != nil || last.Attempts != 1 {
		t.Fatalf("last event = %+v, want the next job waiting for another try", last)
	}
}

func TestUnknownJobIsReported(t *testing.T) {
	h := newHarness(t)
	err := h.runner.Process(context.Background(), jobcontracts.Message{JobID: domain.NewID(), Type: "x", Payload: []byte("{}")})
	if err == nil || errors.Is(err, queue.ErrRetry) {
		t.Fatalf("want a plain error (message acked), got %v", err)
	}
}

func TestProgressIsSavedAndThrottled(t *testing.T) {
	h := newHarness(t)
	var seen []int
	h.runner.Dispatcher.Register(jobcontracts.JobTypeTranscodeHLS, func(ctx context.Context, job jobs.Job) error {
		read := func() int {
			j, _ := h.store.GetJob(ctx, job.ID)
			return j.Progress
		}
		job.Progress(10)
		seen = append(seen, read())
		job.Progress(20) // within a second of the last write: skipped
		seen = append(seen, read())
		time.Sleep(progressInterval + 50*time.Millisecond)
		job.Progress(30)
		seen = append(seen, read())
		return nil
	})

	if err := h.process(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if want := []int{10, 10, 30}; !slices.Equal(seen, want) {
		t.Fatalf("progress in database = %v, want %v", seen, want)
	}
	if got := h.job(2); got.Progress != 100 {
		t.Fatalf("finished job progress = %d, want 100", got.Progress)
	}
}
