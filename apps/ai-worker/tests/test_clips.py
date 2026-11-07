"""Clip suggestions: prompt, answer cleaning and the job end to end with a
fake model and the real database."""

import logging
import threading
import time

import pytest

from fake_llm import FakeLLM
from ultramedia_ai import contracts
from ultramedia_ai.answers import InvalidAnswer, NotConfigured
from ultramedia_ai.clips import ClipSuggester, build_prompt, clean_clips, parse_answer, speech_segments
from ultramedia_ai.llm import ChatClient
from ultramedia_ai.runner import JobContext

# Twelve 10-second segments: two minutes of speech.
SEGMENTS = [{"start": i * 10.0, "end": i * 10.0 + 9.5, "text": f"Frase número {i}."} for i in range(12)]


def test_speech_segments_and_prompt():
    segs = speech_segments(SEGMENTS + [{"start": 120, "end": 125, "text": "  "}, {"start": 130, "end": 130, "text": "x"}])
    assert len(segs) == 12
    prompt = build_prompt("pt", segs)
    assert '"pt"' in prompt and "up to 5 moments" in prompt
    assert "[0.0-9.5] Frase número 0." in prompt and "[110.0-119.5] Frase número 11." in prompt


def test_clean_clips_snaps_filters_and_ranks():
    raw = [
        {"start": 1, "end": 39, "title": "Abertura", "reason": "Começa forte.", "score": 6},  # -> 0.0-39.5
        {"start": "00:31", "end": "01:09", "title": "Quase igual", "score": 9},  # -> 30-69.5
        {"start": 40, "end": 79, "title": "Mesmo trecho", "score": 7},  # mostly inside the score 9 clip
        {"start": 80, "end": 84, "title": "Curto demais", "score": 10},  # end snaps to 79.5: dropped
        {"start": 0, "end": 119.5, "title": "Longo demais", "score": 10},  # 119.5 s
        {"start": 90, "end": 119, "title": '"Final"', "score": "8.6"},  # -> 90-119.5, score 9
        {"start": 50, "end": 20, "title": "Invertido"},
        {"start": 10, "end": 30},  # no title
        "lixo",
    ]
    clips = clean_clips(raw, SEGMENTS)
    assert [(c["rank"], c["start"], c["end"], c["title"], c["score"]) for c in clips] == [
        (1, 30.0, 69.5, "Quase igual", 9),
        (2, 90.0, 119.5, "Final", 9),
        (3, 0.0, 39.5, "Abertura", 6),
    ]
    assert clips[2]["reason"] == "Começa forte." and clips[0]["reason"] == ""


def test_clean_clips_keeps_at_most_five():
    raw = [{"start": i * 10, "end": i * 10 + 19.5, "title": f"C{i}", "score": 5} for i in range(0, 11, 2)]
    assert [c["title"] for c in clean_clips(raw, SEGMENTS)] == ["C0", "C2", "C4", "C6", "C8"]


def test_parse_answer():
    assert parse_answer('{"clips": []}', SEGMENTS) == []
    with pytest.raises(InvalidAnswer, match='"clips"'):
        parse_answer('{"clip": []}', SEGMENTS)


# ---- The job ----


@pytest.fixture
def fake():
    with FakeLLM() as f:
        yield f


def run(cfg, store, asset, fake):
    client = ChatClient(fake.base_url, "sk-test", "ultramedia-text", timeout=5, backoff=0.01) if fake else None
    job_id = asset["jobs"][contracts.JOB_SUGGEST_CLIPS]
    ctx = JobContext(
        job=type("J", (), {"id": job_id})(), log=logging.getLogger("test"),
        deadline=time.monotonic() + 60, stop=threading.Event(), timeout_seconds=60,
    )
    ClipSuggester(cfg, store, client, logging.getLogger("test"))(ctx, f'{{"asset_id": "{asset["id"]}"}}')


def save_transcript(store, asset, segments):
    text = " ".join(s["text"] for s in segments)
    store.save_transcript(asset["id"], asset["jobs"][contracts.JOB_TRANSCRIBE_AUDIO], "pt", text, segments, "base")


def test_saves_clips_and_a_rerun_replaces_them(cfg, store, asset, fake):
    save_transcript(store, asset, SEGMENTS)
    fake.answer({"clips": [{"start": 10, "end": 49.5, "title": "Bom trecho", "reason": "Tem gancho.", "score": 8}]},
                litellm_model="openai/claude-haiku-5-5")
    run(cfg, store, asset, fake)
    assert store.get_clips(asset["id"]) == [{
        "start": 10.0, "end": 49.5, "title": "Bom trecho", "reason": "Tem gancho.", "score": 8, "rank": 1,
        "model": "claude-haiku-5-5",
    }]
    sent = fake.requests[0]["body"]["messages"]
    assert [m["role"] for m in sent] == ["system", "user"] and "[10.0-19.5] Frase número 1." in sent[1]["content"]

    fake.answer({"clips": []})
    run(cfg, store, asset, fake)
    assert store.get_clips(asset["id"]) == []


def test_asks_again_when_the_answer_is_not_json(cfg, store, asset, fake):
    save_transcript(store, asset, SEGMENTS)
    fake.answer("Here are some clips!")
    fake.answer({"clips": [{"start": 0, "end": 29.5, "title": "Início", "score": 7}]})
    run(cfg, store, asset, fake)
    assert [c["title"] for c in store.get_clips(asset["id"])] == ["Início"]


def test_short_or_silent_videos_get_no_clips_without_asking(cfg, store, asset, fake):
    run(cfg, store, asset, fake)  # no transcript
    save_transcript(store, asset, SEGMENTS[:2])  # 19.5 seconds of speech
    run(cfg, store, asset, fake)
    assert fake.requests == [] and store.get_clips(asset["id"]) == []


def test_not_configured(cfg, store, asset):
    save_transcript(store, asset, SEGMENTS)
    with pytest.raises(NotConfigured, match="LLM_BASE_URL"):
        run(cfg, store, asset, None)
