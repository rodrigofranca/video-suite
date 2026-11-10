"""Job lifecycle against the real database (mirrors the Go runner tests)."""

import logging
import threading
import time

import pytest

from ultramedia_ai import contracts, runner as runner_mod
from ultramedia_ai.runner import Retry, Runner


class Harness:
    def __init__(self, store, asset):
        self.store, self.asset = store, asset
        self.enqueued: list[contracts.Message] = []
        self.calls: list[str] = []
        self.behavior = {}
        self.stop = threading.Event()
        handlers = {t: self._handler(t) for t in contracts.PIPELINE}
        self.runner = Runner(store, handlers, self.enqueued.append, logging.getLogger("test"), self.stop, timeout_seconds=5)

    def _handler(self, job_type):
        def handle(ctx, payload):
            self.calls.append(job_type)
            ctx.log.info("working")
            fn = self.behavior.get(job_type)
            if fn:
                fn(ctx)
        return handle

    def process(self, job_type):
        job = self.store.get_job(self.asset["jobs"][job_type])
        return self.runner.process(contracts.Message(job.id, job.type, job.input_json))

    def job(self, job_type):
        return self.store.get_job(self.asset["jobs"][job_type])

    def logs(self, job_type):
        rows = self.store.conn.execute(
            "SELECT level, message FROM job_logs WHERE job_id = %s ORDER BY timestamp, id", (self.asset["jobs"][job_type],)
        ).fetchall()
        return rows


@pytest.fixture
def h(store, asset):
    return Harness(store, asset)


def test_success_chains_next_job(h):
    h.process(contracts.JOB_EXTRACT_AUDIO)
    job = h.job(contracts.JOB_EXTRACT_AUDIO)
    assert (job.status, job.progress) == ("done", 100)
    assert len(h.enqueued) == 1
    nxt = h.enqueued[0]
    assert nxt.job_id == h.asset["jobs"][contracts.JOB_TRANSCRIBE_AUDIO]
    assert nxt.stream == contracts.STREAM_AI
    messages = [m for _, m in h.logs(contracts.JOB_EXTRACT_AUDIO)]
    assert messages[:3] == ["job started", "working", "job finished"]
    assert messages[3].startswith("queued next job next_type=transcribe-audio")


def test_transcription_chains_analysis_on_the_ai_stream(h):
    h.process(contracts.JOB_TRANSCRIBE_AUDIO)
    assert [m.job_id for m in h.enqueued] == [h.asset["jobs"][contracts.JOB_ANALYZE_TRANSCRIPT]]
    assert h.enqueued[0].stream == contracts.STREAM_AI


def test_last_job_does_not_chain(h):
    h.process(contracts.JOB_TRACK_FACES)
    assert h.job(contracts.JOB_TRACK_FACES).status == "done"
    assert h.enqueued == []


@pytest.mark.parametrize("fails", [False, True])
def test_a_translation_queues_the_export_waiting_for_it(h, fails):
    """Even after a failed translation: the export then fails with the reason."""
    ids = {}
    for i, (job_type, payload) in enumerate(
        [(contracts.JOB_TRANSLATE_TRANSCRIPT, '{"language": "en"}'), (contracts.JOB_EXPORT_CLIP, '{"format": "vertical"}')]
    ):
        ids[job_type] = h.store.conn.execute(
            """INSERT INTO processing_jobs (id, asset_id, type, input_json, status, created_at)
               VALUES (gen_random_uuid(), %s, %s, %s, 'pending', NOW() + %s * INTERVAL '1 ms') RETURNING id::text""",
            (h.asset["id"], job_type, payload, i),
        ).fetchone()[0]

    def translate(ctx, payload):
        if fails:
            raise RuntimeError("gateway down")

    h.runner.handlers[contracts.JOB_TRANSLATE_TRANSCRIPT] = translate
    job = h.store.get_job(ids[contracts.JOB_TRANSLATE_TRANSCRIPT])
    try:
        h.runner.process(contracts.Message(job.id, job.type, job.input_json))
    except RuntimeError:
        assert fails
    assert [(m.job_id, m.stream) for m in h.enqueued] == [(ids[contracts.JOB_EXPORT_CLIP], contracts.STREAM_MEDIA)]


def test_model_calls_are_recorded_for_the_job(h):
    from ultramedia_ai.llm import Completion

    def two_calls(ctx):
        ctx.record_usage(Completion("{}", "claude-haiku-5-5", "stop", 1200, 300, 0.00027))
        ctx.record_usage(Completion("{}", "claude-haiku-5-5", "stop", 1500, None, None))  # no usage, no price

    h.behavior[contracts.JOB_ANALYZE_TRANSCRIPT] = two_calls
    h.process(contracts.JOB_ANALYZE_TRANSCRIPT)
    job_id = h.asset["jobs"][contracts.JOB_ANALYZE_TRANSCRIPT]
    rows = h.store.llm_usage(job_id)
    # Usage outlives its video: remove these rows, or they would count in the dev totals.
    asset = h.store.conn.execute("SELECT DISTINCT asset_id::text FROM llm_usage WHERE job_id = %s", (job_id,)).fetchone()[0]
    h.store.conn.execute("DELETE FROM llm_usage WHERE job_id = %s", (job_id,))
    assert rows == [
        {"job_type": "analyze-transcript", "model": "claude-haiku-5-5", "prompt_tokens": 1200, "completion_tokens": 300, "cost_usd": 0.00027},
        {"job_type": "analyze-transcript", "model": "claude-haiku-5-5", "prompt_tokens": 1500, "completion_tokens": 0, "cost_usd": None},
    ]
    assert asset == h.asset["id"]


def test_failure_is_saved_and_reraised(h):
    def boom(ctx):
        raise RuntimeError("model crashed")

    h.behavior[contracts.JOB_TRANSCRIBE_AUDIO] = boom
    with pytest.raises(RuntimeError):
        h.process(contracts.JOB_TRANSCRIBE_AUDIO)
    job = h.job(contracts.JOB_TRANSCRIBE_AUDIO)
    assert (job.status, job.error_message) == ("failed", "model crashed")
    assert ("error", "job failed: model crashed") in h.logs(contracts.JOB_TRANSCRIBE_AUDIO)


def test_duplicate_delivery_is_skipped(h):
    h.process(contracts.JOB_TRANSCRIBE_AUDIO)
    h.process(contracts.JOB_TRANSCRIBE_AUDIO)
    assert h.calls == [contracts.JOB_TRANSCRIBE_AUDIO]


def test_timeout(h):
    h.runner.timeout_seconds = 0.05

    def slow(ctx):
        time.sleep(0.1)
        ctx.check()

    h.behavior[contracts.JOB_TRANSCRIBE_AUDIO] = slow
    with pytest.raises(Exception):
        h.process(contracts.JOB_TRANSCRIBE_AUDIO)
    assert h.job(contracts.JOB_TRANSCRIBE_AUDIO).error_message == "timed out after 0s"


def test_shutdown_leaves_job_to_resume(h):
    def interrupted(ctx):
        h.stop.set()
        ctx.check()

    h.behavior[contracts.JOB_TRANSCRIBE_AUDIO] = interrupted
    with pytest.raises(Retry):
        h.process(contracts.JOB_TRANSCRIBE_AUDIO)
    assert h.job(contracts.JOB_TRANSCRIBE_AUDIO).status == "running"

    # After a restart the same message runs again and completes.
    h.stop.clear()
    del h.behavior[contracts.JOB_TRANSCRIBE_AUDIO]
    h.process(contracts.JOB_TRANSCRIBE_AUDIO)
    assert h.job(contracts.JOB_TRANSCRIBE_AUDIO).status == "done"


def test_progress_is_throttled(h, monkeypatch):
    seen = []

    def progress(ctx):
        read = lambda: h.store.get_job(ctx.job.id).progress  # noqa: E731
        ctx.progress(10)
        seen.append(read())
        ctx.progress(20)  # within a second of the last write: skipped
        seen.append(read())

    monkeypatch.setattr(runner_mod, "PROGRESS_INTERVAL", 1.0)
    h.behavior[contracts.JOB_TRANSCRIBE_AUDIO] = progress
    h.process(contracts.JOB_TRANSCRIBE_AUDIO)
    assert seen == [10, 10]
    assert h.job(contracts.JOB_TRANSCRIBE_AUDIO).progress == 100


def test_enqueue_failure_fails_next_job_and_asset(h):
    def broken(msg):
        raise ConnectionError("redis down")

    h.runner.enqueue = broken
    h.process(contracts.JOB_EXTRACT_AUDIO)
    nxt = h.job(contracts.JOB_TRANSCRIBE_AUDIO)
    assert nxt.status == "failed" and "could not enqueue" in nxt.error_message
    status = h.store.conn.execute("SELECT status FROM media_assets WHERE id = %s", (h.asset["id"],)).fetchone()[0]
    assert status == "failed"


def test_unknown_job(h):
    with pytest.raises(LookupError):
        h.runner.process(contracts.Message("00000000-0000-4000-8000-000000000000", "x", "{}"))
