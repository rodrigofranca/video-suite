"""Consumer behavior against a real Redis (mirrors the Go queue tests)."""

import logging
import threading
import time
import uuid

from ultramedia_ai import contracts
from ultramedia_ai.queue import Consumer
from ultramedia_ai.runner import Retry, UnsupportedJobType


class Recorder:
    def __init__(self):
        self.jobs = []

    def __call__(self, msg):
        self.jobs.append(msg.job_id)
        if msg.type == "fails":
            raise RuntimeError("boom")
        if msg.type == "retry":
            raise Retry("database down")
        if msg.type == "unknown":
            raise UnsupportedJobType(msg.type)


def make(redis_client, name="test-ai"):
    stream = f"test:jobs:ai:{uuid.uuid4()}"
    rec, stop = Recorder(), threading.Event()
    c = Consumer(redis_client, rec, logging.getLogger("test"), name, stop, stream=stream, group="ai-workers", block_ms=50)
    return c, rec, stop


def add(redis_client, stream, job_id, job_type):
    redis_client.xadd(stream, contracts.Message(job_id, job_type, "{}").values())


def run_until(consumer, stop, cond, timeout=5.0):
    stop.clear()
    t = threading.Thread(target=consumer.run)
    t.start()
    deadline = time.monotonic() + timeout
    while not cond():
        assert time.monotonic() < deadline, "timed out waiting for the consumer"
        time.sleep(0.02)
    stop.set()
    t.join(2)
    assert not t.is_alive(), "consumer did not stop"


def pending(redis_client, c):
    return redis_client.xpending(c.stream, c.group)["pending"]


def test_acks_every_message(redis_client):
    c, rec, stop = make(redis_client)
    try:
        for job_id, t in [("j1", "ok"), ("j2", "fails"), ("j3", "unknown"), ("j4", "ok")]:
            add(redis_client, c.stream, job_id, t)
        redis_client.xadd(c.stream, {"type": "ok"})  # malformed
        run_until(c, stop, lambda: len(rec.jobs) == 4 and pending(redis_client, c) == 0)
        assert rec.jobs == ["j1", "j2", "j3", "j4"]
    finally:
        redis_client.delete(c.stream)


def test_retry_stays_pending_and_resumes_once(redis_client):
    c, rec, stop = make(redis_client)
    try:
        add(redis_client, c.stream, "j1", "retry")
        add(redis_client, c.stream, "j2", "ok")
        run_until(c, stop, lambda: len(rec.jobs) == 2)
        assert pending(redis_client, c) == 1
        run_until(c, stop, lambda: len(rec.jobs) == 3)  # restart: j1 again, once
        assert rec.jobs == ["j1", "j2", "j1"]
    finally:
        redis_client.delete(c.stream)
