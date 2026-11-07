"""Transcription end to end: synthetic speech, a real Whisper model (tiny by
default in tests), the real database and storage."""

import logging
import shutil
import subprocess
import threading
import time
import urllib.request

import pytest

from ultramedia_ai import contracts
from ultramedia_ai.runner import JobContext
from ultramedia_ai.transcribe import Transcriber

SENTENCE = "Hello and welcome to the ultra media platform. This video was transcribed automatically."


def speech_flac(path, text=SENTENCE):
    """Synthesize speech with FFmpeg's flite filter, as 16 kHz mono FLAC."""
    if not shutil.which("ffmpeg"):
        pytest.skip("ffmpeg not installed")
    filters = subprocess.run(["ffmpeg", "-hide_banner", "-filters"], capture_output=True, text=True).stdout
    if " flite " not in filters:
        pytest.skip("ffmpeg was built without the flite filter")
    subprocess.run(
        ["ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", f"flite=text='{text}':voice=slt",
         "-ac", "1", "-ar", "16000", "-c:a", "flac", str(path)],
        check=True,
    )


class Ctx:
    def __init__(self, job_id):
        self.progress_values = []
        self.ctx = JobContext(
            job=type("J", (), {"id": job_id})(),
            log=logging.getLogger("test"),
            deadline=time.monotonic() + 300,
            stop=threading.Event(),
            timeout_seconds=300,
            _report=self.progress_values.append,
        )


@pytest.fixture
def transcriber(cfg, storage, store):
    return Transcriber(cfg, storage, store, logging.getLogger("test"))


def test_transcribes_speech_and_publishes_subtitles(cfg, storage, store, asset, transcriber, tmp_path):
    key = contracts.audio_key(asset["id"])
    speech_flac(tmp_path / "speech.flac")
    storage.upload_file(key, str(tmp_path / "speech.flac"), "audio/flac")
    vtt_key = contracts.subtitles_key(asset["id"], "en")
    try:
        c = Ctx(asset["jobs"][contracts.JOB_TRANSCRIBE_AUDIO])
        transcriber(c.ctx, f'{{"asset_id": "{asset["id"]}", "audio_key": "{key}"}}')

        t = store.get_transcript(asset["id"])
        assert t["language"] == "en" and t["model"] == cfg.whisper_model
        assert "welcome" in t["text"].lower() and "platform" in t["text"].lower()
        assert t["segments"] and t["segments"][0]["end"] > 0
        assert c.progress_values and all(0 <= p <= 99 for p in c.progress_values)

        # Subtitles are public, like the HLS output.
        public = f"{cfg.s3_endpoint}/{cfg.s3_bucket}/{vtt_key}"
        with urllib.request.urlopen(public) as resp:
            body = resp.read().decode()
            assert resp.headers["Content-Type"].startswith("text/vtt")
        assert body.startswith("WEBVTT") and "-->" in body and "welcome" in body.lower()
        row = store.conn.execute(
            "SELECT storage_key, metadata_json->>'language' FROM artifacts WHERE asset_id = %s AND type = 'subtitles'",
            (asset["id"],),
        ).fetchone()
        assert row == (vtt_key, "en")
    finally:
        storage.delete(key)
        storage.delete(vtt_key)


def test_no_audio_is_not_an_error(storage, store, asset, transcriber):
    key = contracts.audio_key(asset["id"])
    c = Ctx(asset["jobs"][contracts.JOB_TRANSCRIBE_AUDIO])
    transcriber(c.ctx, f'{{"asset_id": "{asset["id"]}", "audio_key": "{key}"}}')
    assert store.get_transcript(asset["id"]) is None


def test_rejects_foreign_keys(transcriber, asset):
    c = Ctx(asset["jobs"][contracts.JOB_TRANSCRIBE_AUDIO])
    with pytest.raises(ValueError, match="invalid input"):
        transcriber(c.ctx, '{"asset_id": "%s", "audio_key": "assets/other/audio/audio.flac"}' % asset["id"])
