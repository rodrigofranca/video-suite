"""Configuration from environment variables (same names as the Go services)."""

from __future__ import annotations

import os
import socket
from dataclasses import dataclass


class ConfigError(Exception):
    pass


@dataclass(frozen=True)
class Config:
    redis_url: str
    database_url: str
    s3_endpoint: str
    s3_region: str
    s3_access_key: str
    s3_secret_key: str
    s3_bucket: str
    worker_name: str
    whisper_model: str
    whisper_device: str
    whisper_compute_type: str
    temp_dir: str | None
    job_timeout_seconds: float


def from_env(env: dict[str, str] | None = None) -> Config:
    env = dict(os.environ if env is None else env)
    required = ["REDIS_URL", "DATABASE_URL", "S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"]
    missing = [name for name in required if not env.get(name)]
    if missing:
        raise ConfigError("missing environment variables: " + ", ".join(missing))
    return Config(
        redis_url=env["REDIS_URL"],
        database_url=env["DATABASE_URL"],
        s3_endpoint=env["S3_ENDPOINT"],
        s3_region=env.get("S3_REGION") or "us-east-1",
        s3_access_key=env["S3_ACCESS_KEY"],
        s3_secret_key=env["S3_SECRET_KEY"],
        s3_bucket=env["S3_BUCKET"],
        # A stable name lets a restarted worker pick up the job it was running.
        worker_name=env.get("AI_WORKER_NAME") or socket.gethostname(),
        # "base" balances speed and quality on a CPU; "small" or "medium" are
        # more accurate and slower. Models download on first use.
        whisper_model=env.get("WHISPER_MODEL") or "base",
        whisper_device=env.get("WHISPER_DEVICE") or "cpu",
        whisper_compute_type=env.get("WHISPER_COMPUTE_TYPE") or "int8",
        temp_dir=env.get("WORKER_TEMP_DIR") or None,
        job_timeout_seconds=30 * 60,  # same limit as the Go worker (CONTEXT.md)
    )
