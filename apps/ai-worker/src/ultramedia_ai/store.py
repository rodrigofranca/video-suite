"""SQL used by the AI worker. It mirrors packages/media-core/store (Go)."""

from __future__ import annotations

import json
import uuid
from dataclasses import dataclass
from datetime import datetime

import psycopg
from psycopg.types.json import Jsonb

JOB_PENDING, JOB_RUNNING, JOB_DONE, JOB_FAILED = "pending", "running", "done", "failed"
ASSET_FAILED = "failed"
ARTIFACT_SUBTITLES = "subtitles"


@dataclass
class Job:
    id: str
    asset_id: str
    type: str
    input_json: str
    status: str
    progress: int
    error_message: str | None
    created_at: datetime


_JOB_COLUMNS = "id::text, asset_id::text, type, input_json::text, status, progress, error_message, created_at"


def _job(row) -> Job | None:
    return Job(*row) if row else None


class Store:
    def __init__(self, conn: psycopg.Connection):
        self.conn = conn

    @classmethod
    def connect(cls, database_url: str) -> "Store":
        # Autocommit: every statement stands alone, like the Go pool.
        return cls(psycopg.connect(database_url, autocommit=True, connect_timeout=5))

    def close(self) -> None:
        self.conn.close()

    # ---- Jobs ----

    def get_job(self, job_id: str) -> Job | None:
        row = self.conn.execute(f"SELECT {_JOB_COLUMNS} FROM processing_jobs WHERE id = %s", (job_id,)).fetchone()
        return _job(row)

    def start_job(self, job_id: str) -> bool:
        """Mark a job running. False when it already finished (duplicate delivery)."""
        cur = self.conn.execute(
            """UPDATE processing_jobs
               SET status = %s, started_at = NOW(), completed_at = NULL, error_message = NULL, progress = 0
               WHERE id = %s AND status IN (%s, %s)""",
            (JOB_RUNNING, job_id, JOB_PENDING, JOB_RUNNING),
        )
        return cur.rowcount == 1

    def finish_job(self, job_id: str, error: str | None) -> None:
        if error is None:
            self.conn.execute(
                "UPDATE processing_jobs SET status = %s, progress = 100, error_message = NULL, completed_at = NOW() WHERE id = %s",
                (JOB_DONE, job_id),
            )
        else:
            self.conn.execute(
                "UPDATE processing_jobs SET status = %s, error_message = %s, completed_at = NOW() WHERE id = %s",
                (JOB_FAILED, error, job_id),
            )

    def set_progress(self, job_id: str, percent: int) -> None:
        self.conn.execute(
            "UPDATE processing_jobs SET progress = %s WHERE id = %s AND status = %s",
            (max(0, min(int(percent), 100)), job_id, JOB_RUNNING),
        )

    def next_pending_job(self, asset_id: str, job_type: str) -> Job | None:
        row = self.conn.execute(
            f"""SELECT {_JOB_COLUMNS} FROM processing_jobs
                WHERE asset_id = %s AND type = %s AND status = %s ORDER BY created_at LIMIT 1""",
            (asset_id, job_type, JOB_PENDING),
        ).fetchone()
        return _job(row)

    def add_log(self, job_id: str, level: str, message: str) -> None:
        self.conn.execute(
            "INSERT INTO job_logs (id, job_id, level, message, timestamp) VALUES (%s, %s, %s, %s, NOW())",
            (str(uuid.uuid4()), job_id, level, message),
        )

    def set_asset_status(self, asset_id: str, status: str) -> None:
        self.conn.execute("UPDATE media_assets SET status = %s, updated_at = NOW() WHERE id = %s", (status, asset_id))

    # ---- Outputs ----

    def create_artifact(self, asset_id: str, job_id: str, artifact_type: str, storage_key: str, metadata: dict) -> None:
        self.conn.execute(
            """INSERT INTO artifacts (id, asset_id, job_id, type, storage_key, metadata_json)
               VALUES (%s, %s, %s, %s, %s, %s)""",
            (str(uuid.uuid4()), asset_id, job_id, artifact_type, storage_key, Jsonb(metadata)),
        )

    def save_transcript(
        self, asset_id: str, job_id: str, language: str, text: str, segments: list[dict], model: str,
        words: list[dict] | None = None,
    ) -> None:
        """Store an asset's transcript, replacing any earlier one. words holds
        each word's start and end, for word-by-word captions."""
        self.conn.execute(
            """INSERT INTO transcripts (id, asset_id, job_id, language, text, segments, words, model)
               VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
               ON CONFLICT (asset_id) DO UPDATE
               SET id = EXCLUDED.id, job_id = EXCLUDED.job_id, language = EXCLUDED.language,
                   text = EXCLUDED.text, segments = EXCLUDED.segments, words = EXCLUDED.words,
                   model = EXCLUDED.model, created_at = NOW()""",
            (str(uuid.uuid4()), asset_id, job_id, language, text, Jsonb(segments), Jsonb(words or []), model),
        )

    def get_transcript(self, asset_id: str) -> dict | None:
        row = self.conn.execute(
            "SELECT language, text, segments, model, words FROM transcripts WHERE asset_id = %s", (asset_id,)
        ).fetchone()
        if not row:
            return None
        segments = row[2] if not isinstance(row[2], str) else json.loads(row[2])
        words = row[4] if not isinstance(row[4], str) else json.loads(row[4])
        return {"language": row[0], "text": row[1], "segments": segments, "model": row[3], "words": words}

    def save_analysis(
        self, asset_id: str, job_id: str, language: str, summary: str, tags: list[str], chapters: list[dict], model: str
    ) -> None:
        """Store an asset's analysis, replacing any earlier one (see SaveAnalysis in Go)."""
        self.conn.execute(
            """INSERT INTO analyses (id, asset_id, job_id, language, summary, tags, chapters, model)
               VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
               ON CONFLICT (asset_id) DO UPDATE
               SET id = EXCLUDED.id, job_id = EXCLUDED.job_id, language = EXCLUDED.language,
                   summary = EXCLUDED.summary, tags = EXCLUDED.tags, chapters = EXCLUDED.chapters,
                   model = EXCLUDED.model, created_at = NOW()""",
            (str(uuid.uuid4()), asset_id, job_id, language, summary, list(tags), Jsonb(chapters), model),
        )

    def get_analysis(self, asset_id: str) -> dict | None:
        row = self.conn.execute(
            "SELECT language, summary, tags, chapters, model FROM analyses WHERE asset_id = %s", (asset_id,)
        ).fetchone()
        if not row:
            return None
        chapters = row[3] if not isinstance(row[3], str) else json.loads(row[3])
        return {"language": row[0], "summary": row[1], "tags": list(row[2]), "chapters": chapters, "model": row[4]}

    def replace_clips(self, asset_id: str, job_id: str, clips: list[dict], model: str) -> None:
        """Store an asset's suggested clips, removing earlier ones (see ReplaceClips in Go)."""
        with self.conn.transaction():
            self.conn.execute("DELETE FROM clips WHERE asset_id = %s", (asset_id,))
            for c in clips:
                self.conn.execute(
                    """INSERT INTO clips (id, asset_id, job_id, start_seconds, end_seconds, title, reason, score, rank, model)
                       VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
                    (str(uuid.uuid4()), asset_id, job_id, c["start"], c["end"], c["title"], c["reason"],
                     c["score"], c["rank"], model),
                )

    def get_clips(self, asset_id: str) -> list[dict]:
        rows = self.conn.execute(
            """SELECT start_seconds, end_seconds, title, reason, score, rank, model
               FROM clips WHERE asset_id = %s ORDER BY rank""",
            (asset_id,),
        ).fetchall()
        keys = ("start", "end", "title", "reason", "score", "rank", "model")
        return [dict(zip(keys, r)) for r in rows]
