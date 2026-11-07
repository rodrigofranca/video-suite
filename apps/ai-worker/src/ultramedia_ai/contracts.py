"""Job contracts shared with the Go code (packages/job-contracts).

The Go package is the source of truth. tests/test_contracts.py checks these
values against packages/job-contracts/testdata/contracts.json, so a change on
one side that is not mirrored on the other fails a test suite.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass
from typing import Mapping

JOB_EXTRACT_METADATA = "extract-metadata"
JOB_GENERATE_THUMBNAIL = "generate-thumbnail"
JOB_TRANSCODE_HLS = "transcode-hls"
JOB_EXTRACT_AUDIO = "extract-audio"
JOB_TRANSCRIBE_AUDIO = "transcribe-audio"
JOB_ANALYZE_TRANSCRIPT = "analyze-transcript"
JOB_SUGGEST_CLIPS = "suggest-clips"
JOB_TRACK_FACES = "track-faces"

# Order in which an upload's jobs run (see jobcontracts.Pipeline).
PIPELINE = [
    JOB_EXTRACT_METADATA,
    JOB_GENERATE_THUMBNAIL,
    JOB_TRANSCODE_HLS,
    JOB_EXTRACT_AUDIO,
    JOB_TRANSCRIBE_AUDIO,
    JOB_ANALYZE_TRANSCRIPT,
    JOB_SUGGEST_CLIPS,
    JOB_TRACK_FACES,
]

STREAM_MEDIA = "jobs"  # Go media worker
STREAM_AI = "jobs:ai"  # this worker
CONSUMER_GROUPS = {STREAM_MEDIA: "workers", STREAM_AI: "ai-workers"}

FIELD_JOB_ID = "job_id"
FIELD_TYPE = "type"
FIELD_PAYLOAD = "payload"

AI_JOB_TYPES = {JOB_TRANSCRIBE_AUDIO, JOB_ANALYZE_TRANSCRIPT, JOB_SUGGEST_CLIPS, JOB_TRACK_FACES}

_UUID = re.compile(r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")


def is_id(value: str) -> bool:
    """Whether value looks like a UUID (see domain.IsID)."""
    return bool(_UUID.match(value or ""))


def stream_for(job_type: str) -> str:
    """The stream a job type is queued on (see jobcontracts.StreamFor)."""
    return STREAM_AI if job_type in AI_JOB_TYPES else STREAM_MEDIA


def next_job_type(job_type: str) -> str | None:
    """The job type that follows job_type in the pipeline, if any."""
    try:
        i = PIPELINE.index(job_type)
    except ValueError:
        return None
    return PIPELINE[i + 1] if i + 1 < len(PIPELINE) else None


@dataclass(frozen=True)
class Message:
    """One entry of a jobs stream. The job row in PostgreSQL is the source of truth."""

    job_id: str
    type: str
    payload: str  # JSON of the typed input

    def values(self) -> dict[str, str]:
        return {FIELD_JOB_ID: self.job_id, FIELD_TYPE: self.type, FIELD_PAYLOAD: self.payload}

    @property
    def stream(self) -> str:
        return stream_for(self.type)

    @classmethod
    def parse(cls, values: Mapping[str, str]) -> "Message":
        missing = [f for f in (FIELD_JOB_ID, FIELD_TYPE, FIELD_PAYLOAD) if not values.get(f)]
        if missing:
            raise ValueError("missing field(s): " + ", ".join(missing))
        payload = values[FIELD_PAYLOAD]
        try:
            json.loads(payload)
        except json.JSONDecodeError as e:
            raise ValueError(f"field {FIELD_PAYLOAD!r} is not valid JSON") from e
        return cls(job_id=values[FIELD_JOB_ID], type=values[FIELD_TYPE], payload=payload)


@dataclass(frozen=True)
class TranscribeAudioInput:
    """See jobcontracts.TranscribeAudioInput. An empty language means detect it."""

    asset_id: str
    audio_key: str
    language: str = ""

    @classmethod
    def from_json(cls, payload: str | bytes) -> "TranscribeAudioInput":
        data = json.loads(payload)
        return cls(
            asset_id=str(data.get("asset_id", "")),
            audio_key=str(data.get("audio_key", "")),
            language=str(data.get("language") or ""),
        )


@dataclass(frozen=True)
class AnalyzeTranscriptInput:
    """See jobcontracts.AnalyzeTranscriptInput. The transcript is read from the database."""

    asset_id: str

    @classmethod
    def from_json(cls, payload: str | bytes) -> "AnalyzeTranscriptInput":
        data = json.loads(payload)
        return cls(asset_id=str(data.get("asset_id", "")))


@dataclass(frozen=True)
class SuggestClipsInput:
    """See jobcontracts.SuggestClipsInput."""

    asset_id: str

    @classmethod
    def from_json(cls, payload: str | bytes) -> "SuggestClipsInput":
        data = json.loads(payload)
        return cls(asset_id=str(data.get("asset_id", "")))


@dataclass(frozen=True)
class TrackFacesInput:
    """See jobcontracts.TrackFacesInput."""

    asset_id: str
    storage_key: str

    @classmethod
    def from_json(cls, payload: str | bytes) -> "TrackFacesInput":
        data = json.loads(payload)
        return cls(asset_id=str(data.get("asset_id", "")), storage_key=str(data.get("storage_key", "")))


def source_key(asset_id: str) -> str:
    """See storage.SourceKey in packages/media-core."""
    return f"assets/{asset_id}/source/input.mp4"


def audio_key(asset_id: str) -> str:
    """See storage.AudioKey in packages/media-core."""
    return f"assets/{asset_id}/audio/audio.flac"


def subtitles_key(asset_id: str, language: str) -> str:
    """See storage.SubtitlesKey in packages/media-core."""
    return f"assets/{asset_id}/subtitles/{language}.vtt"
