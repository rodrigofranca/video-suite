"""Runs one job end to end. It mirrors apps/worker/internal/pipeline (Go):
status in PostgreSQL, logs in job_logs, a time limit, progress, and the next
pipeline step."""

from __future__ import annotations

import logging
import threading
import time
from dataclasses import dataclass, field
from typing import Callable

from . import contracts
from .store import ASSET_FAILED, Job, Store


class Retry(Exception):
    """Do not ack the message: it runs again when the worker restarts."""


class Interrupted(Exception):
    """The worker is shutting down."""


class JobTimeout(Exception):
    pass


class UnsupportedJobType(Exception):
    pass


PROGRESS_INTERVAL = 1.0  # seconds between two progress writes


@dataclass
class JobContext:
    """What a handler receives besides the payload."""

    job: Job
    log: logging.Logger
    deadline: float  # time.monotonic() limit
    stop: threading.Event
    timeout_seconds: float
    _report: Callable[[int], None] = field(repr=False, default=lambda p: None)

    def progress(self, percent: int) -> None:
        self._report(percent)

    def check(self) -> None:
        """Call between steps of long work: stops on shutdown or timeout."""
        if self.stop.is_set():
            raise Interrupted()
        if time.monotonic() > self.deadline:
            raise JobTimeout(f"timed out after {_duration(self.timeout_seconds)}")


Handler = Callable[[JobContext, str], None]


class Runner:
    def __init__(
        self,
        store: Store,
        handlers: dict[str, Handler],
        enqueue: Callable[[contracts.Message], None],
        log: logging.Logger,
        stop: threading.Event,
        timeout_seconds: float = 30 * 60,
    ):
        self.store = store
        self.handlers = handlers
        self.enqueue = enqueue
        self.log = log
        self.stop = stop
        self.timeout_seconds = timeout_seconds

    def process(self, msg: contracts.Message) -> None:
        try:
            job = self.store.get_job(msg.job_id)
        except Exception as e:  # database unreachable: try again after restart
            raise Retry(f"load job: {e}") from e
        if job is None:
            raise LookupError(f"job {msg.job_id} not found in database")
        joblog = self._job_logger(job)

        try:
            started = self.store.start_job(job.id)
        except Exception as e:
            raise Retry(f"start job: {e}") from e
        if not started:
            # Delivered twice: do not run again, but make sure the pipeline moved on.
            self.log.info("job already finished, skipping job_id=%s status=%s", job.id, job.status)
            self._advance(job, joblog)
            return

        joblog.info("job started")
        ctx = JobContext(
            job=job,
            log=joblog,
            deadline=time.monotonic() + self.timeout_seconds,
            stop=self.stop,
            timeout_seconds=self.timeout_seconds,
            _report=self._progress_reporter(job.id),
        )
        error: Exception | None = None
        try:
            handler = self.handlers.get(job.type)
            if handler is None:
                raise UnsupportedJobType(f"unsupported job type: {job.type}")
            handler(ctx, job.input_json)
        except Interrupted:
            joblog.warning("interrupted by worker shutdown; it will run again on restart")
            raise Retry("interrupted by shutdown")
        except Exception as e:
            error = e

        if error is not None:
            joblog.error("job failed: %s", error)
        else:
            joblog.info("job finished")
        try:
            self.store.finish_job(job.id, str(error) if error is not None else None)
        except Exception:
            self.log.exception("could not save job result job_id=%s", job.id)
        self._advance(job, joblog)
        if error is not None:
            raise error

    def _advance(self, job: Job, joblog: logging.Logger) -> None:
        """Enqueue the next pending job of the asset (see Runner.advance in Go)."""
        next_type = contracts.next_job_type(job.type)
        if next_type is None:
            return
        try:
            nxt = self.store.next_pending_job(job.asset_id, next_type)
        except Exception as e:
            joblog.error("could not load next job: %s", e)
            return
        if nxt is None:
            return
        try:
            self.enqueue(contracts.Message(job_id=nxt.id, type=nxt.type, payload=nxt.input_json))
        except Exception as e:
            joblog.error("could not enqueue %s: %s", nxt.type, e)
            self.store.finish_job(nxt.id, f"could not enqueue job: {e}")
            self.store.set_asset_status(job.asset_id, ASSET_FAILED)
            return
        joblog.info("queued next job next_type=%s next_job_id=%s", nxt.type, nxt.id)

    def _progress_reporter(self, job_id: str) -> Callable[[int], None]:
        last = {"value": -1, "at": 0.0}

        def report(percent: int) -> None:
            now = time.monotonic()
            if percent == last["value"] or now - last["at"] < PROGRESS_INTERVAL:
                return
            try:
                self.store.set_progress(job_id, percent)
            except Exception as e:
                self.log.warning("could not save job progress job_id=%s: %s", job_id, e)
                return
            last.update(value=percent, at=now)

        return report

    def _job_logger(self, job: Job) -> logging.Logger:
        """A logger that writes to job_logs and to the worker output."""
        logger = logging.Logger(f"job.{job.id}", level=logging.INFO)  # not registered: freed after the job
        logger.addHandler(JobLogHandler(self.store, job.id, self.log))
        for h in self.log.handlers or logging.getLogger().handlers:
            logger.addHandler(_Prefixed(h, f"job_id={job.id} type={job.type} "))
        return logger


class JobLogHandler(logging.Handler):
    """Saves each record as a job_logs row."""

    LEVELS = {logging.WARNING: "warning", logging.ERROR: "error", logging.CRITICAL: "error"}

    def __init__(self, store: Store, job_id: str, fallback: logging.Logger):
        super().__init__(level=logging.INFO)
        self.store, self.job_id, self.fallback = store, job_id, fallback

    def emit(self, record: logging.LogRecord) -> None:
        try:
            self.store.add_log(self.job_id, self.LEVELS.get(record.levelno, "info"), record.getMessage())
        except Exception as e:
            self.fallback.warning("could not save job log job_id=%s: %s", self.job_id, e)


class _Prefixed(logging.Handler):
    """Forwards records to another handler with the job identified."""

    def __init__(self, target: logging.Handler, prefix: str):
        super().__init__(level=logging.INFO)
        self.target, self.prefix = target, prefix

    def emit(self, record: logging.LogRecord) -> None:
        copy = logging.makeLogRecord(record.__dict__)
        copy.msg, copy.args = self.prefix + record.getMessage(), None
        self.target.handle(copy)


def _duration(seconds: float) -> str:
    """Formats like Go's time.Duration (30m0s), matching the Go worker's messages."""
    m, s = divmod(int(seconds), 60)
    h, m = divmod(m, 60)
    return (f"{h}h" if h else "") + (f"{m}m" if h or m else "") + f"{s}s"
