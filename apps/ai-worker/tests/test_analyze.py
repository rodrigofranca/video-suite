"""Transcript analysis: prompt, answer parsing and the job end to end with a
fake model and the real database."""

import logging
import threading
import time
from dataclasses import replace

import pytest

from fake_llm import FakeLLM
from ultramedia_ai import contracts
from ultramedia_ai.analyze import (
    Analyzer,
    InvalidAnswer,
    NotConfigured,
    Passage,
    build_prompt,
    clean_chapters,
    clean_tags,
    clock,
    extract_json,
    merge_passages,
    parse_answer,
    parse_time,
)
from ultramedia_ai.llm import ChatClient, LLMError
from ultramedia_ai.runner import JobContext

SEGMENTS = [
    {"start": 0.0, "end": 8.0, "text": " Olá e bem-vindos."},
    {"start": 8.0, "end": 21.0, "text": "Hoje vamos falar de vídeo."},
    {"start": 21.0, "end": 30.0, "text": "   "},
    {"start": 30.5, "end": 45.0, "text": "Primeiro, como funciona o HLS."},
    {"start": 60.0, "end": 75.0, "text": "Depois, legendas."},
]


# ---- Prompt ----


def test_merge_passages():
    passages = merge_passages(SEGMENTS)
    assert passages == [
        Passage(0.0, "Olá e bem-vindos. Hoje vamos falar de vídeo."),
        Passage(30.5, "Primeiro, como funciona o HLS. Depois, legendas."),
    ]
    assert merge_passages([]) == [] and merge_passages([{"start": 0, "end": 1, "text": " "}]) == []


def test_prompt_has_language_length_and_timestamps():
    prompt = build_prompt("pt", 75.0, merge_passages(SEGMENTS))
    assert '"pt"' in prompt and "The speech lasts 01:15" in prompt
    assert "[00:00] Olá e bem-vindos." in prompt and "[00:30] Primeiro" in prompt


def test_clock():
    assert [clock(5), clock(65.9), clock(3725)] == ["00:05", "01:05", "1:02:05"]


@pytest.mark.parametrize(
    "value,want",
    [(12, 12.0), (1.5, 1.5), ("01:05", 65.0), ("1:02:05", 3725.0), ("90", 90.0), ("1:75", None), ("x", None), (True, None), (None, None)],
)
def test_parse_time(value, want):
    assert parse_time(value) == want


# ---- Answer ----


@pytest.mark.parametrize(
    "text",
    [
        '{"summary": "ok"}',
        '```json\n{"summary": "ok"}\n```',
        '<think>reasoning {not json}</think>\n{"summary": "ok"}',
        'Here it is: {"summary": "ok"} Hope it helps.',
    ],
)
def test_extract_json(text):
    assert extract_json(text) == {"summary": "ok"}


@pytest.mark.parametrize("text,error", [("", "empty"), ("no json here", "no JSON"), ("{bad json}", "not valid"), ("[1, 2]", "not a JSON object")])
def test_extract_json_errors(text, error):
    with pytest.raises(InvalidAnswer, match=error):
        extract_json(text)


def test_clean_tags():
    assert clean_tags(["Vídeo", "#hls", " vídeo ", "", 3, "x" * 41, "legendas  automáticas"]) == [
        "Vídeo", "hls", "legendas automáticas"
    ]
    assert clean_tags("a, b") == ["a", "b"] and clean_tags(None) == []
    assert len(clean_tags([f"t{i}" for i in range(20)])) == 10


def test_clean_chapters_snaps_sorts_and_spaces():
    starts = [0.0, 30.5, 60.0, 65.0, 120.0]
    raw = [
        {"start": "02:00", "title": "Fim"},
        {"start": 2, "title": '"Abertura"'},  # first chapter moves to 0
        {"start": "00:31", "title": "HLS"},  # snapped to 30.5
        {"start": 64, "title": "Too close"},  # snapped to 65, under 10 s after 60
        {"start": "01:00", "title": "Legendas"},
        {"start": 500, "title": "After the end"},
        {"start": "1:00", "title": ""},
        "nonsense",
    ]
    assert clean_chapters(raw, starts, end=150.0) == [
        {"start": 0.0, "title": "Abertura"},
        {"start": 30.5, "title": "HLS"},
        {"start": 60.0, "title": "Legendas"},
        {"start": 120.0, "title": "Fim"},
    ]


def test_one_chapter_is_no_chapters():
    assert clean_chapters([{"start": 0, "title": "Tudo"}], [0.0, 30.0], 60.0) == []
    assert clean_chapters("x", [0.0], 60.0) == [] and clean_chapters([], [], 60.0) == []


def test_parse_answer_requires_a_summary():
    with pytest.raises(InvalidAnswer, match="summary"):
        parse_answer('{"tags": ["a"]}', [0.0], 10.0)
    result = parse_answer('{"summary": "  Um vídeo. ", "tags": "a,b", "chapters": []}', [0.0], 10.0)
    assert (result.summary, result.tags, result.chapters) == ("Um vídeo.", ["a", "b"], [])


# ---- The job ----

GOOD = {
    "summary": "Uma introdução ao HLS e às legendas.",
    "tags": ["hls", "legendas"],
    "chapters": [{"start": "00:00", "title": "Abertura"}, {"start": "00:30", "title": "HLS"}],
}


@pytest.fixture
def fake():
    with FakeLLM() as f:
        yield f


def make_ctx(job_id, timeout=60):
    return JobContext(
        job=type("J", (), {"id": job_id})(),
        log=logging.getLogger("test"),
        deadline=time.monotonic() + timeout,
        stop=threading.Event(),
        timeout_seconds=timeout,
    )


def analyzer(cfg, store, fake, **cfg_changes):
    client = ChatClient(fake.base_url, "sk-test", "ultramedia-text", timeout=5, backoff=0.01) if fake else None
    return Analyzer(replace(cfg, **cfg_changes), store, client, logging.getLogger("test"))


def run(a, asset):
    job_id = asset["jobs"][contracts.JOB_ANALYZE_TRANSCRIPT]
    a(make_ctx(job_id), f'{{"asset_id": "{asset["id"]}"}}')


def save_transcript(store, asset, segments=SEGMENTS, language="pt"):
    text = " ".join(s["text"].strip() for s in segments if s["text"].strip())
    store.save_transcript(asset["id"], asset["jobs"][contracts.JOB_TRANSCRIBE_AUDIO], language, text, segments, "base")


def test_saves_summary_tags_and_chapters(cfg, store, asset, fake):
    save_transcript(store, asset)
    fake.answer(GOOD, model="deepseek-v4-flash")
    run(analyzer(cfg, store, fake), asset)
    assert store.get_analysis(asset["id"]) == {
        "language": "pt",
        "summary": GOOD["summary"],
        "tags": ["hls", "legendas"],
        "chapters": [{"start": 0.0, "title": "Abertura"}, {"start": 30.5, "title": "HLS"}],
        "model": "deepseek-v4-flash",
    }
    sent = fake.requests[0]["body"]
    assert sent["model"] == "ultramedia-text"
    assert [m["role"] for m in sent["messages"]] == ["system", "user"]
    assert "[00:30] Primeiro, como funciona o HLS." in sent["messages"][1]["content"]


def test_asks_again_once_when_the_answer_is_not_json(cfg, store, asset, fake):
    save_transcript(store, asset)
    fake.answer("Sure! Here is a summary of the video.")
    fake.answer("```json\n" + '{"summary": "Agora sim.", "tags": [], "chapters": []}' + "\n```")
    run(analyzer(cfg, store, fake), asset)
    assert store.get_analysis(asset["id"])["summary"] == "Agora sim."
    repair = fake.requests[1]["body"]["messages"]
    assert [m["role"] for m in repair] == ["system", "user", "assistant", "user"]
    assert "could not be used: the answer has no JSON object" in repair[-1]["content"]


def test_fails_after_two_unusable_answers(cfg, store, asset, fake):
    save_transcript(store, asset)
    fake.answer("nope")
    fake.answer('{"tags": []}')
    with pytest.raises(InvalidAnswer, match='could not be used: "summary" is missing'):
        run(analyzer(cfg, store, fake), asset)
    assert store.get_analysis(asset["id"]) is None


def test_a_rerun_replaces_the_analysis(cfg, store, asset, fake):
    save_transcript(store, asset)
    fake.answer(GOOD)
    fake.answer({"summary": "Segunda versão.", "tags": ["novo"], "chapters": []})
    a = analyzer(cfg, store, fake)
    run(a, asset)
    run(a, asset)
    got = store.get_analysis(asset["id"])
    assert (got["summary"], got["tags"], got["chapters"]) == ("Segunda versão.", ["novo"], [])


def test_nothing_to_analyze(cfg, store, asset, fake):
    run(analyzer(cfg, store, fake), asset)  # no transcript at all (no audio track)
    save_transcript(store, asset, segments=[], language="en")  # silence
    run(analyzer(cfg, store, fake), asset)
    assert fake.requests == [] and store.get_analysis(asset["id"]) is None


def test_gateway_errors_fail_the_job(cfg, store, asset, fake):
    save_transcript(store, asset)
    fake.fail(401, "invalid key")
    with pytest.raises(LLMError, match="401: invalid key"):
        run(analyzer(cfg, store, fake), asset)


def test_too_long_and_not_configured(cfg, store, asset, fake):
    save_transcript(store, asset)
    with pytest.raises(ValueError, match="too long to analyze.*LLM_MAX_INPUT_CHARS"):
        run(analyzer(cfg, store, fake, llm_max_input_chars=500), asset)
    with pytest.raises(NotConfigured, match="LLM_BASE_URL"):
        run(analyzer(cfg, store, None), asset)
    assert fake.requests == []


def test_rejects_bad_input(cfg, store, fake):
    with pytest.raises(ValueError, match="invalid input"):
        analyzer(cfg, store, fake)(make_ctx("x"), '{"asset_id": "../etc"}')
