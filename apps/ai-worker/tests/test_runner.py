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
        self.calls: list[str] = []
        self.behavior = {}
        self.stop = threading.Event()
        handlers = {t: self._handler(t) for t in contracts.PIPELINE}
        self.runner = Runner(store, handlers, logging.getLogger("test"), self.stop, timeout_seconds=5)

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

    def events(self, job_type):
        """The outbox events of a job: what the relay will act on (docs/adr/0034)."""
        return self.store.conn.execute(
            "SELECT type, handled_at FROM outbox_events WHERE job_id = %s ORDER BY id", (self.asset["jobs"][job_type],)
        ).fetchall()

    def logs(self, job_type):
        rows = self.store.conn.execute(
            "SELECT level, message FROM job_logs WHERE job_id = %s ORDER BY timestamp, id", (self.asset["jobs"][job_type],)
        ).fetchall()
        return rows


@pytest.fixture
def h(store, asset):
    return Harness(store, asset)


def test_success_records_the_finished_event(h):
    h.process(contracts.JOB_EXTRACT_AUDIO)
    job = h.job(contracts.JOB_EXTRACT_AUDIO)
    assert (job.status, job.progress) == ("done", 100)
    # The relay in the Go processes queues the next job from this event.
    assert h.events(contracts.JOB_EXTRACT_AUDIO) == [(contracts.EVENT_JOB_FINISHED, None)]
    messages = [m for _, m in h.logs(contracts.JOB_EXTRACT_AUDIO)]
    assert messages == ["job started", "working", "job finished"]


@pytest.mark.parametrize("fails", [False, True])
def test_a_failed_job_records_the_event_too(h, fails):
    """The pipeline goes on after a failure: an export waiting for a failed
    translation then fails with the reason."""
    def crash(ctx):
        raise RuntimeError("model crashed")

    if fails:
        h.behavior[contracts.JOB_TRANSCRIBE_AUDIO] = crash
    try:
        h.process(contracts.JOB_TRANSCRIBE_AUDIO)
    except RuntimeError:
        assert fails
    assert h.job(contracts.JOB_TRANSCRIBE_AUDIO).status == ("failed" if fails else "done")
    assert h.events(contracts.JOB_TRANSCRIBE_AUDIO) == [(contracts.EVENT_JOB_FINISHED, None)]


def test_an_unknown_job_records_nothing(store):
    store.finish_job("00000000-0000-4000-8000-000000000000", None)
    n = store.conn.execute(
        "SELECT COUNT(*) FROM outbox_events WHERE job_id = '00000000-0000-4000-8000-000000000000'"
    ).fetchone()[0]
    assert n == 0


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
    assert len(h.events(contracts.JOB_TRANSCRIBE_AUDIO)) == 1  # the pipeline moves on once


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


def test_unknown_job(h):
    with pytest.raises(LookupError):
        h.runner.process(contracts.Message("00000000-0000-4000-8000-000000000000", "x", "{}"))
