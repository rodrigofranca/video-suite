"""The transcribe-audio job: speech to text with faster-whisper, plus WebVTT
subtitles for the player."""

from __future__ import annotations

import logging
import os
import tempfile
import threading

from . import contracts, subtitles
from .config import Config
from .runner import JobContext
from .storage import Storage
from .store import ARTIFACT_SUBTITLES, Store


class Transcriber:
    def __init__(self, cfg: Config, storage: Storage, store: Store, log: logging.Logger):
        self.cfg, self.storage, self.store, self.log = cfg, storage, store, log
        self._model = None
        self._lock = threading.Lock()

    def load_model(self):
        """Load the Whisper model, downloading it on first use (cached in HF_HOME)."""
        with self._lock:
            if self._model is None:
                from faster_whisper import WhisperModel  # heavy import, only when needed

                self.log.info(
                    "loading Whisper model %s (%s, %s); the first run downloads it",
                    self.cfg.whisper_model, self.cfg.whisper_device, self.cfg.whisper_compute_type,
                )
                self._model = WhisperModel(
                    self.cfg.whisper_model, device=self.cfg.whisper_device, compute_type=self.cfg.whisper_compute_type
                )
            return self._model

    def __call__(self, ctx: JobContext, payload: str) -> None:
        inp = contracts.TranscribeAudioInput.from_json(payload)
        if not contracts.is_id(inp.asset_id) or inp.audio_key != contracts.audio_key(inp.asset_id):
            raise ValueError(f"invalid input: asset_id {inp.asset_id!r}, audio_key {inp.audio_key!r}")

        if not self.storage.exists(inp.audio_key):
            ctx.log.info("no audio to transcribe: the video has no audio track")
            return

        model = self.load_model()
        with tempfile.TemporaryDirectory(prefix=f"transcribe-{inp.asset_id}-", dir=self.cfg.temp_dir) as tmp:
            path = os.path.join(tmp, "audio.flac")
            ctx.log.info("downloading audio key=%s", inp.audio_key)
            self.storage.download_file(inp.audio_key, path)

            ctx.log.info("transcribing with Whisper %s, language %s", self.cfg.whisper_model, inp.language or "auto")
            segments_iter, info = model.transcribe(
                path,
                language=inp.language or None,
                vad_filter=True,  # skip silence: faster, and fewer invented words
                word_timestamps=True,  # lets subtitles split into short cues
            )
            ctx.log.info(
                "language %s (probability %.2f), %.1fs of audio",
                info.language, info.language_probability, info.duration,
            )
            segments: list[subtitles.Segment] = []
            for s in segments_iter:  # decoded lazily, segment by segment
                ctx.check()
                words = [subtitles.Word(w.start, w.end, w.word) for w in (s.words or [])]
                segments.append(subtitles.Segment(s.start, s.end, s.text.strip(), words))
                if info.duration > 0:
                    ctx.progress(min(99, int(s.end / info.duration * 100)))

        language = inp.language or info.language
        text = " ".join(s.text for s in segments if s.text).strip()
        self.store.save_transcript(
            inp.asset_id, ctx.job.id, language, text,
            [{"start": round(s.start, 3), "end": round(s.end, 3), "text": s.text} for s in segments],
            self.cfg.whisper_model,
        )
        if not segments:
            ctx.log.info("no speech found")
            return

        cues = subtitles.build_cues(segments)
        key = contracts.subtitles_key(inp.asset_id, language)
        self.storage.upload_text(key, subtitles.to_webvtt(cues), "text/vtt; charset=utf-8")
        self.store.create_artifact(
            inp.asset_id, ctx.job.id, ARTIFACT_SUBTITLES, key, {"language": language, "cues": len(cues)}
        )
        ctx.log.info(
            "transcribed %d segments, %d words; subtitles key=%s cues=%d",
            len(segments), len(text.split()), key, len(cues),
        )
