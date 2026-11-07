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


def test_last_job_does_not_chain(h):
    h.process(contracts.JOB_TRANSCRIBE_AUDIO)
    assert h.job(contracts.JOB_TRANSCRIBE_AUDIO).status == "done"
    assert h.enqueued == []


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
