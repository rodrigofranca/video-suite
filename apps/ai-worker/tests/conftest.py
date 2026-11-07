"""Shared fixtures. Tests that need PostgreSQL, Redis or storage use the
services from `make up && make migrate` and are skipped when those are down."""

from __future__ import annotations

import json
import os
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path

import pytest
import redis

from ultramedia_ai import contracts
from ultramedia_ai.config import from_env
from ultramedia_ai.storage import Storage
from ultramedia_ai.store import Store

GOLDEN = Path(__file__).resolve().parents[3] / "packages" / "job-contracts" / "testdata" / "contracts.json"

DEFAULTS = {
    "REDIS_URL": "redis://localhost:6379",
    "DATABASE_URL": "postgres://ultramedia:ultramedia@localhost:5432/ultramedia?sslmode=disable",
    "S3_ENDPOINT": "http://localhost:9000",
    "S3_ACCESS_KEY": "ultramedia",
    "S3_SECRET_KEY": "ultramedia",
    "S3_BUCKET": "ultramedia",
}


@pytest.fixture(scope="session")
def golden() -> dict:
    return json.loads(GOLDEN.read_text())


@pytest.fixture(scope="session")
def cfg():
    env = {**DEFAULTS, **{k: v for k, v in os.environ.items() if k in DEFAULTS or k.startswith("WHISPER_")}}
    env.setdefault("WHISPER_MODEL", "tiny")  # small and fast enough for tests
    return from_env(env)


@pytest.fixture
def store(cfg):
    try:
        s = Store.connect(cfg.database_url)
        s.conn.execute("SELECT 1 FROM clips LIMIT 1")  # the newest table: all migrations applied
    except Exception as e:  # noqa: BLE001
        pytest.skip(f"database not available or not migrated: {e}")
    yield s
    s.close()


@pytest.fixture
def redis_client(cfg):
    client = redis.Redis.from_url(cfg.redis_url, decode_responses=True, socket_connect_timeout=1)
    try:
        client.ping()
    except redis.RedisError as e:
        pytest.skip(f"redis not available: {e}")
    yield client
    client.close()


@pytest.fixture
def storage(cfg):
    s = Storage(cfg)
    try:
        s.check()
    except Exception as e:  # noqa: BLE001
        pytest.skip(f"storage not available: {e}")
    return s


@pytest.fixture
def asset(store):
    """An asset with one pending job per pipeline step, removed afterwards."""
    asset_id = str(uuid.uuid4())
    store.conn.execute(
        "INSERT INTO media_assets (id, filename, status) VALUES (%s, %s, 'ready')", (asset_id, "speech.mp4")
    )
    store.conn.execute(
        """INSERT INTO media_sources (id, asset_id, storage_key, mime_type, size_bytes, checksum)
           VALUES (%s, %s, %s, 'video/mp4', 1, 'x')""",
        (str(uuid.uuid4()), asset_id, f"assets/{asset_id}/source/input.mp4"),
    )
    jobs = {}
    base = datetime.now(timezone.utc)
    for i, job_type in enumerate(contracts.PIPELINE):
        job_id = str(uuid.uuid4())
        payload = json.dumps({"asset_id": asset_id, "audio_key": contracts.audio_key(asset_id)})
        store.conn.execute(
            """INSERT INTO processing_jobs (id, asset_id, type, input_json, status, created_at)
               VALUES (%s, %s, %s, %s, 'pending', %s)""",
            (job_id, asset_id, job_type, payload, base + timedelta(milliseconds=i)),
        )
        jobs[job_type] = job_id
    yield {"id": asset_id, "jobs": jobs}
    store.conn.execute("DELETE FROM media_assets WHERE id = %s", (asset_id,))
