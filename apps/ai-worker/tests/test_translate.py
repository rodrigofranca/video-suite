"""Transcript translation: batches, prompt, answer parsing, word placement
and the job end to end with a fake model, the real database and storage."""

import json
import logging
from dataclasses import replace

import pytest

from fake_llm import FakeLLM, demo_answer
from test_analyze import make_ctx
from ultramedia_ai import contracts, translate
from ultramedia_ai.answers import InvalidAnswer, NotConfigured
from ultramedia_ai.llm import ChatClient
from ultramedia_ai.translate import (
    Translator,
    build_prompt,
    make_batches,
    parse_answer,
    place_all,
    place_words,
)

SEGMENTS = [
    {"start": 0.0, "end": 2.0, "text": " Olá pessoal."},
    {"start": 2.0, "end": 3.0, "text": "   "},
    {"start": 3.0, "end": 7.0, "text": "Hoje vamos falar de legendas."},
]
WORDS = [
    {"start": 0.2, "end": 0.6, "word": "Olá"},
    {"start": 0.7, "end": 1.6, "word": "pessoal."},
    {"start": 3.0, "end": 3.4, "word": "Hoje"},
    {"start": 3.4, "end": 3.8, "word": "vamos"},
    {"start": 3.8, "end": 4.2, "word": "falar"},
    {"start": 4.2, "end": 4.4, "word": "de"},
    # A pause, then the last word.
    {"start": 6.0, "end": 6.8, "word": "legendas."},
]


# ---- Prompt ----


def test_batches_keep_lines_in_order_within_limits(monkeypatch):
    monkeypatch.setattr(translate, "BATCH_LINES", 3)
    monkeypatch.setattr(translate, "BATCH_CHARS", 12)
    lines = [{"text": t} for t in ["a" * 5, "b" * 5, "c" * 5, "d", "e", "f", "g"]]
    batches = make_batches(lines)
    assert [(first, "".join(s["text"][0] for s in b)) for first, b in batches] == [(0, "ab"), (2, "cde"), (5, "fg")]
    assert make_batches([]) == []
    # One line longer than a batch still goes, alone.
    assert [len(b) for _, b in make_batches([{"text": "x" * 50}, {"text": "y"}])] == [1, 1]


def test_prompt_names_languages_numbers_lines_and_gives_context():
    prompt = build_prompt("pt", "en", ["Olá pessoal.", "Tudo bem?"], ["Antes disso."])
    assert prompt.startswith("Translate these subtitle lines of a video from Brazilian Portuguese into English.")
    assert "exactly 2 strings" in prompt
    assert "1. Olá pessoal.\n2. Tudo bem?" in prompt
    assert "for context only (do not translate them):\n- Antes disso." in prompt
    assert "context only" not in build_prompt("en", "es", ["Hi."], [])
    assert 'the language with ISO 639-1 code "ja"' in build_prompt("ja", "en", ["こんにちは"], [])


def test_the_demo_model_marks_each_line_with_the_language():
    prompt = build_prompt("pt", "es", ["Olá.", "Tudo bem?"], ["- não conta"])
    body = {"messages": [{"role": "user", "content": prompt}]}
    answer = json.loads(demo_answer(body)[1]["choices"][0]["message"]["content"])
    assert answer == {"lines": ["(Spanish) Olá.", "(Spanish) Tudo bem?"]}


# ---- Answer ----


def test_parse_answer():
    assert parse_answer('```json\n{"lines": ["Hello  everyone.", "2. Fine?"]}\n```', 2) == ["Hello everyone.", "Fine?"]
    assert parse_answer('{"lines": [{"n": 1, "text": "Hi."}]}', 1) == ["Hi."]
    # Only the line's own number is taken for a copied numbering.
    assert parse_answer('{"lines": ["3. Three people came."]}', 1) == ["3. Three people came."]


@pytest.mark.parametrize(
    "text,error",
    [
        ('{"text": "Hi"}', '"lines" is missing'),
        ('{"lines": ["Hi."]}', '"lines" has 1 strings; there were 2'),
        ('{"lines": ["Hi.", "  "]}', "line 2 is empty"),
        ('{"lines": ["Hi.", 3]}', "line 2 is empty"),
    ],
)
def test_parse_answer_rejects(text, error):
    with pytest.raises(InvalidAnswer, match=error):
        parse_answer(text, 2)


# ---- Word timings ----


def test_words_follow_the_original_speech_and_its_pauses():
    said = WORDS[2:]
    words = place_words("Today we will talk about captions.", 3.0, 7.0, said)
    assert [w["word"] for w in words] == ["Today", "we", "will", "talk", "about", "captions."]
    starts = [w["start"] for w in words]
    assert starts == sorted(starts) and words[0]["start"] == 3.0 and words[-1]["end"] == 6.8
    # "captions." sits where "legendas." was said, after the pause.
    assert words[-1]["start"] >= 6.0
    # No word is placed inside the pause.
    assert all(not (4.4 < w["start"] < 6.0) for w in words)


def test_without_word_timings_the_line_is_shared_by_length():
    words = place_words("aaa b", 10.0, 12.0, [])
    assert words == [{"start": 10.0, "end": 11.333, "word": "aaa"}, {"start": 11.333, "end": 12.0, "word": "b"}]
    assert place_words("   ", 0.0, 1.0, []) == []


def test_place_all_uses_each_lines_own_words_in_order():
    lines = [{"start": 0.0, "end": 2.0, "text": "Hi everyone."}, {"start": 3.0, "end": 7.0, "text": "Today, captions."}]
    words = place_all(lines, WORDS)
    assert [w["word"] for w in words] == ["Hi", "everyone.", "Today,", "captions."]
    assert words[0]["start"] == 0.2 and words[1]["end"] == 1.6
    assert words[2]["start"] == 3.0 and words[3]["end"] == 6.8
    assert all(a["end"] <= b["start"] for a, b in zip(words, words[1:]))


# ---- The job ----


@pytest.fixture
def fake():
    with FakeLLM() as f:
        yield f


def translator(cfg, storage, store, fake, **cfg_changes):
    client = ChatClient(fake.base_url, "sk-test", "ultramedia-text", timeout=5, backoff=0.01) if fake else None
    return Translator(replace(cfg, **cfg_changes), storage, store, client, logging.getLogger("test"))


def run(t, asset, language="en"):
    job_id = asset["jobs"][contracts.JOB_ANALYZE_TRANSCRIPT]  # any job row of the asset
    t(make_ctx(job_id), json.dumps({"asset_id": asset["id"], "language": language}))
    return job_id


def save_transcript(store, asset, segments=SEGMENTS, words=WORDS, language="pt"):
    text = " ".join(s["text"].strip() for s in segments if s["text"].strip())
    store.save_transcript(asset["id"], asset["jobs"][contracts.JOB_TRANSCRIBE_AUDIO], language, text, segments, "base", words=words)


def test_translates_and_saves_words_and_subtitles(cfg, storage, store, asset, fake, tmp_path):
    save_transcript(store, asset)
    fake.answer({"lines": ["Hi everyone.", "Today we will talk about captions."]}, model="claude-haiku-5-5")
    job_id = run(translator(cfg, storage, store, fake), asset)

    got = store.get_translation(asset["id"], "en")
    assert got["source_language"] == "pt" and got["model"] == "claude-haiku-5-5"
    assert got["segments"] == [
        {"start": 0.0, "end": 2.0, "text": "Hi everyone."},
        {"start": 3.0, "end": 7.0, "text": "Today we will talk about captions."},
    ]
    assert [w["word"] for w in got["words"]] == ["Hi", "everyone.", "Today", "we", "will", "talk", "about", "captions."]

    sent = fake.requests[0]["body"]
    assert sent["model"] == "ultramedia-text" and sent["max_tokens"] == translate.MAX_TOKENS
    assert "from Brazilian Portuguese into English" in sent["messages"][1]["content"]
    assert "1. Olá pessoal.\n2. Hoje vamos falar de legendas." in sent["messages"][1]["content"]

    storage.download_file(contracts.subtitles_key(asset["id"], "en"), str(tmp_path / "en.vtt"))
    vtt = (tmp_path / "en.vtt").read_text()
    assert vtt.startswith("WEBVTT") and "Hi everyone." in vtt and "captions." in vtt
    row = store.conn.execute(
        "SELECT storage_key, metadata_json FROM artifacts WHERE job_id = %s AND type = 'subtitles'", (job_id,)
    ).fetchone()
    assert row[0] == f"assets/{asset['id']}/subtitles/en.vtt"
    assert row[1]["language"] == "en" and row[1]["translatedFrom"] == "pt" and row[1]["cues"] >= 2


def test_long_transcripts_go_in_several_requests_with_context(cfg, storage, store, asset, fake, monkeypatch):
    monkeypatch.setattr(translate, "BATCH_LINES", 2)
    segments = [{"start": float(i), "end": i + 0.9, "text": f"Frase {i}."} for i in range(5)]
    save_transcript(store, asset, segments=segments, words=[])
    fake.default = demo_answer
    run(translator(cfg, storage, store, fake), asset, "es")
    got = store.get_translation(asset["id"], "es")
    assert [s["text"] for s in got["segments"]] == [f"(Spanish) Frase {i}." for i in range(5)]
    prompts = [r["body"]["messages"][1]["content"] for r in fake.requests]
    assert len(prompts) == 3
    assert "- Frase 0.\n- Frase 1." in prompts[1] and "1. Frase 2.\n2. Frase 3." in prompts[1]
    # Without word timings, each line's time is shared by its words.
    assert got["words"][0] == {"start": 0.0, "end": 0.474, "word": "(Spanish)"}  # 10 of 19 characters of 0.9 s


def test_asks_again_when_lines_are_missing(cfg, storage, store, asset, fake):
    save_transcript(store, asset)
    fake.answer({"lines": ["Hi everyone. Today we will talk about captions."]})
    fake.answer({"lines": ["Hi everyone.", "Today, captions."]})
    run(translator(cfg, storage, store, fake), asset)
    assert [s["text"] for s in store.get_translation(asset["id"], "en")["segments"]] == ["Hi everyone.", "Today, captions."]
    assert "has 1 strings; there were 2 lines" in fake.requests[1]["body"]["messages"][-1]["content"]


def test_a_new_transcript_removes_the_translations(cfg, storage, store, asset, fake):
    save_transcript(store, asset)
    fake.answer({"lines": ["Hi everyone.", "Today, captions."]})
    run(translator(cfg, storage, store, fake), asset)
    save_transcript(store, asset)
    assert store.get_translation(asset["id"], "en") is None


def test_refuses_what_cannot_be_translated(cfg, storage, store, asset, fake):
    t = translator(cfg, storage, store, fake)
    with pytest.raises(ValueError, match="no transcript"):
        run(t, asset)
    save_transcript(store, asset)
    with pytest.raises(ValueError, match="already in Brazilian Portuguese"):
        run(t, asset, "pt")
    with pytest.raises(ValueError, match="invalid input"):
        run(t, asset, "ja")
    save_transcript(store, asset, segments=[{"start": 0, "end": 1, "text": " "}], words=[])
    with pytest.raises(ValueError, match="no speech"):
        run(t, asset)
    with pytest.raises(NotConfigured, match="LLM_BASE_URL"):
        run(translator(cfg, storage, store, None), asset)
    assert fake.requests == []
