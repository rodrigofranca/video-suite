"""SQL used by the AI worker. It mirrors packages/media-core/store (Go)."""

from __future__ import annotations

import json
import uuid
from dataclasses import dataclass
from datetime import datetime

import psycopg
from psycopg.types.json import Jsonb

from . import contracts

JOB_PENDING, JOB_RUNNING, JOB_DONE, JOB_FAILED = "pending", "running", "done", "failed"
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
        """Save a job's result and record the job.finished event that moves the
        pipeline on (docs/adr/0034), in one transaction."""
        with self.conn.transaction():
            if error is None:
                row = self.conn.execute(
                    """UPDATE processing_jobs SET status = %s, progress = 100, error_message = NULL, completed_at = NOW()
                       WHERE id = %s RETURNING asset_id""",
                    (JOB_DONE, job_id),
                ).fetchone()
            else:
                row = self.conn.execute(
                    "UPDATE processing_jobs SET status = %s, error_message = %s, completed_at = NOW() WHERE id = %s RETURNING asset_id",
                    (JOB_FAILED, error, job_id),
                ).fetchone()
            if row is None:
                return  # an unknown job is left alone, as in Go
            self.conn.execute(
                "INSERT INTO outbox_events (type, asset_id, job_id) VALUES (%s, %s, %s)",
                (contracts.EVENT_JOB_FINISHED, row[0], job_id),
            )

    def set_progress(self, job_id: str, percent: int) -> None:
        self.conn.execute(
            "UPDATE processing_jobs SET progress = %s WHERE id = %s AND status = %s",
            (max(0, min(int(percent), 100)), job_id, JOB_RUNNING),
        )

    def add_log(self, job_id: str, level: str, message: str) -> None:
        self.conn.execute(
            "INSERT INTO job_logs (id, job_id, level, message, timestamp) VALUES (%s, %s, %s, %s, NOW())",
            (str(uuid.uuid4()), job_id, level, message),
        )

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
        """Store an asset's transcript, replacing any earlier one and removing
        the translations of the earlier one. words holds each word's start and
        end, for word-by-word captions."""
        with self.conn.transaction():
            self.conn.execute(
                """INSERT INTO transcripts (id, asset_id, job_id, language, text, segments, words, model)
                   VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                   ON CONFLICT (asset_id) DO UPDATE
                   SET id = EXCLUDED.id, job_id = EXCLUDED.job_id, language = EXCLUDED.language,
                       text = EXCLUDED.text, segments = EXCLUDED.segments, words = EXCLUDED.words,
                       model = EXCLUDED.model, created_at = NOW()""",
                (str(uuid.uuid4()), asset_id, job_id, language, text, Jsonb(segments), Jsonb(words or []), model),
            )
            # Translations of the earlier transcript no longer match the speech.
            self.conn.execute("DELETE FROM transcript_translations WHERE asset_id = %s", (asset_id,))

    def get_transcript(self, asset_id: str) -> dict | None:
        row = self.conn.execute(
            "SELECT language, text, segments, model, words FROM transcripts WHERE asset_id = %s", (asset_id,)
        ).fetchone()
        if not row:
            return None
        segments = row[2] if not isinstance(row[2], str) else json.loads(row[2])
        words = row[4] if not isinstance(row[4], str) else json.loads(row[4])
        return {"language": row[0], "text": row[1], "segments": segments, "model": row[3], "words": words}

    def save_translation(
        self, asset_id: str, job_id: str, language: str, source_language: str, segments: list[dict],
        words: list[dict], model: str,
    ) -> None:
        """Store a translation of an asset's transcript, replacing an earlier
        one into the same language (see SaveTranslation in Go)."""
        self.conn.execute(
            """INSERT INTO transcript_translations (id, asset_id, job_id, language, source_language, segments, words, model)
               VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
               ON CONFLICT (asset_id, language) DO UPDATE
               SET id = EXCLUDED.id, job_id = EXCLUDED.job_id, source_language = EXCLUDED.source_language,
                   segments = EXCLUDED.segments, words = EXCLUDED.words, model = EXCLUDED.model, created_at = NOW()""",
            (str(uuid.uuid4()), asset_id, job_id, language, source_language, Jsonb(segments), Jsonb(words), model),
        )

    def get_translation(self, asset_id: str, language: str) -> dict | None:
        row = self.conn.execute(
            """SELECT source_language, segments, words, model FROM transcript_translations
               WHERE asset_id = %s AND language = %s""",
            (asset_id, language),
        ).fetchone()
        if not row:
            return None
        segments = row[1] if not isinstance(row[1], str) else json.loads(row[1])
        words = row[2] if not isinstance(row[2], str) else json.loads(row[2])
        return {"source_language": row[0], "segments": segments, "words": words, "model": row[3]}

    def add_llm_usage(
        self, asset_id: str, job_id: str, job_type: str, model: str,
        prompt_tokens: int, completion_tokens: int, cost_usd: float | None,
    ) -> None:
        """Keep one call to the language model (docs/adr/0032)."""
        self.conn.execute(
            """INSERT INTO llm_usage (id, asset_id, job_id, job_type, model, prompt_tokens, completion_tokens, cost_usd)
               VALUES (%s, %s, %s, %s, %s, %s, %s, %s)""",
            (str(uuid.uuid4()), asset_id, job_id, job_type, model, int(prompt_tokens), int(completion_tokens), cost_usd),
        )

    def llm_usage(self, job_id: str) -> list[dict]:
        """For tests: a job's recorded calls, oldest first."""
        rows = self.conn.execute(
            """SELECT job_type, model, prompt_tokens, completion_tokens, cost_usd FROM llm_usage
               WHERE job_id = %s ORDER BY created_at, id""",
            (job_id,),
        ).fetchall()
        keys = ("job_type", "model", "prompt_tokens", "completion_tokens", "cost_usd")
        return [dict(zip(keys, (r[0], r[1], r[2], r[3], float(r[4]) if r[4] is not None else None))) for r in rows]

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

    def edited_clips(self, asset_id: str) -> list[dict]:
        """The asset's clips changed by hand, which new suggestions keep (docs/adr/0026)."""
        rows = self.conn.execute(
            """SELECT id::text, start_seconds, end_seconds, title, score FROM clips
               WHERE asset_id = %s AND edited_at IS NOT NULL ORDER BY rank""",
            (asset_id,),
        ).fetchall()
        return [{"id": r[0], "start": r[1], "end": r[2], "title": r[3], "score": r[4]} for r in rows]

    def replace_clips(
        self, asset_id: str, job_id: str, clips: list[dict], model: str, edited_ranks: dict[str, int] | None = None
    ) -> None:
        """Store an asset's suggested clips, removing the earlier suggestions
        but not the clips edited by hand, which get the ranks in edited_ranks
        (see ReplaceClips in Go)."""
        with self.conn.transaction():
            self.conn.execute("DELETE FROM clips WHERE asset_id = %s AND edited_at IS NULL", (asset_id,))
            for clip_id, rank in (edited_ranks or {}).items():
                self.conn.execute("UPDATE clips SET rank = %s WHERE id = %s AND asset_id = %s", (rank, clip_id, asset_id))
            for c in clips:
                self.conn.execute(
                    """INSERT INTO clips (id, asset_id, job_id, start_seconds, end_seconds, title, reason, score, rank, model)
                       VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
                    (str(uuid.uuid4()), asset_id, job_id, c["start"], c["end"], c["title"], c["reason"],
                     c["score"], c["rank"], model),
                )

    def mark_edited(self, clip_id: str) -> None:
        """For tests: as if someone had changed the clip by hand."""
        self.conn.execute("UPDATE clips SET edited_at = NOW() WHERE id = %s", (clip_id,))

    def get_clips(self, asset_id: str) -> list[dict]:
        rows = self.conn.execute(
            """SELECT start_seconds, end_seconds, title, reason, score, rank, model
               FROM clips WHERE asset_id = %s ORDER BY rank""",
            (asset_id,),
        ).fetchall()
        keys = ("start", "end", "title", "reason", "score", "rank", "model")
        return [dict(zip(keys, r)) for r in rows]

    def clip_ranges(self, asset_id: str) -> list[dict]:
        """The asset's clips with their id, start and end, best first."""
        rows = self.conn.execute(
            "SELECT id::text, start_seconds, end_seconds FROM clips WHERE asset_id = %s ORDER BY rank", (asset_id,)
        ).fetchall()
        return [{"id": r[0], "start": r[1], "end": r[2]} for r in rows]

    def set_face_track(self, clip_id: str, track: dict | None) -> None:
        self.conn.execute(
            "UPDATE clips SET face_track = %s WHERE id = %s", (Jsonb(track) if track is not None else None, clip_id)
        )

    def get_face_track(self, clip_id: str) -> dict | None:
        row = self.conn.execute("SELECT face_track FROM clips WHERE id = %s", (clip_id,)).fetchone()
        if not row or row[0] is None:
            return None
        return row[0] if not isinstance(row[0], str) else json.loads(row[0])
