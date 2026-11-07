"""Consumes the jobs:ai Redis stream. It mirrors apps/worker/internal/queue (Go):
one message at a time, ack even on failure, except when a job asks to be
retried after a restart."""

from __future__ import annotations

import logging
import threading
import time
from typing import Callable

import redis

from . import contracts
from .runner import Retry, UnsupportedJobType


class Consumer:
    def __init__(
        self,
        client: redis.Redis,
        process: Callable[[contracts.Message], None],
        log: logging.Logger,
        name: str,
        stop: threading.Event,
        stream: str = contracts.STREAM_AI,
        group: str | None = None,
        block_ms: int = 2000,
    ):
        self.r = client
        self.process = process
        self.log = log
        self.name = name
        self.stop = stop
        self.stream = stream
        self.group = group or contracts.CONSUMER_GROUPS[stream]
        self.block_ms = block_ms

    def ensure_group(self) -> None:
        try:
            self.r.xgroup_create(self.stream, self.group, id="0", mkstream=True)
        except redis.ResponseError as e:
            if "BUSYGROUP" not in str(e):
                raise

    def run(self) -> None:
        """Consume until stop is set: first messages left unacked by an earlier
        run of this consumer (for example after a crash), then new ones."""
        self.ensure_group()
        self.recover_pending()
        while not self.stop.is_set():
            try:
                self.read_once()
            except redis.RedisError as e:
                if self.stop.is_set():
                    break
                self.log.error("read from stream failed, retrying: %s", e)
                time.sleep(1)

    def recover_pending(self) -> None:
        after = "0"  # page by ID so a message left unacked on purpose is read once
        while not self.stop.is_set():
            resp = self.r.xreadgroup(self.group, self.name, {self.stream: after}, count=1)
            entries = resp[0][1] if resp else []
            if not entries:
                return
            msg_id, fields = entries[0]
            after = msg_id
            self.log.info("resuming unfinished job message_id=%s", msg_id)
            self.handle(msg_id, fields)

    def read_once(self) -> int:
        resp = self.r.xreadgroup(self.group, self.name, {self.stream: ">"}, count=1, block=self.block_ms)
        n = 0
        for _, entries in resp or []:
            for msg_id, fields in entries:
                n += 1
                self.handle(msg_id, fields)
        return n

    def handle(self, msg_id: str, fields: dict | None) -> None:
        try:
            msg = contracts.Message.parse(fields or {})
        except ValueError as e:
            self.log.error("invalid job message message_id=%s: %s", msg_id, e)
        else:
            try:
                self.process(msg)
            except Retry as e:
                self.log.warning(
                    "job not acknowledged; it will run again after restart message_id=%s job_id=%s reason=%s",
                    msg_id, msg.job_id, e,
                )
                return
            except UnsupportedJobType:
                self.log.warning("unsupported job type: %s", msg.type)
            except Exception as e:  # noqa: BLE001 - one bad job must not stop the worker
                self.log.error("job failed message_id=%s job_id=%s type=%s: %s", msg_id, msg.job_id, msg.type, e)
            else:
                self.log.info("job finished message_id=%s job_id=%s type=%s", msg_id, msg.job_id, msg.type)
        self.r.xack(self.stream, self.group, msg_id)
