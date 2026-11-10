"""The Python contracts must match the Go ones (packages/job-contracts)."""

import json

import pytest

from ultramedia_ai import contracts


def test_constants_match_go(golden):
    assert contracts.PIPELINE == golden["pipeline"]
    assert contracts.STREAM_MEDIA == golden["streams"]["media"]
    assert contracts.STREAM_AI == golden["streams"]["ai"]
    assert contracts.CONSUMER_GROUPS == golden["consumerGroups"]
    assert [contracts.FIELD_JOB_ID, contracts.FIELD_TYPE, contracts.FIELD_PAYLOAD] == [
        golden["fields"]["jobId"], golden["fields"]["type"], golden["fields"]["payload"]
    ]


def test_lists_match_go(golden):
    assert contracts.AFTER == golden["after"]
    assert [contracts.EVENT_JOB_QUEUED, contracts.EVENT_JOB_FINISHED, contracts.OUTBOX_CHANNEL] == [
        golden["events"]["jobQueued"], golden["events"]["jobFinished"], golden["events"]["channel"]
    ]
    assert contracts.CAPTION_LANGUAGES == golden["captionLanguages"]


def test_routing_matches_go(golden):
    for job_type, stream in golden["routes"].items():
        assert contracts.stream_for(job_type) == stream, job_type


def test_transcribe_input_matches_go(golden):
    example = golden["examples"][contracts.JOB_TRANSCRIBE_AUDIO]
    inp = contracts.TranscribeAudioInput.from_json(json.dumps(example))
    assert (inp.asset_id, inp.audio_key, inp.language) == (example["asset_id"], example["audio_key"], "pt")
    assert inp.audio_key == contracts.audio_key(inp.asset_id)
    assert golden["examples"][contracts.JOB_EXTRACT_AUDIO]["audio_key"] == contracts.audio_key(inp.asset_id)


def test_analyze_input_matches_go(golden):
    example = golden["examples"][contracts.JOB_ANALYZE_TRANSCRIPT]
    assert contracts.AnalyzeTranscriptInput.from_json(json.dumps(example)).asset_id == example["asset_id"]
    assert set(example) == {"asset_id"}


def test_suggest_clips_input_matches_go(golden):
    example = golden["examples"][contracts.JOB_SUGGEST_CLIPS]
    assert contracts.SuggestClipsInput.from_json(json.dumps(example)).asset_id == example["asset_id"]
    assert set(example) == {"asset_id"}


def test_track_faces_input_matches_go(golden):
    example = golden["examples"][contracts.JOB_TRACK_FACES]
    inp = contracts.TrackFacesInput.from_json(json.dumps(example))
    assert (inp.asset_id, inp.storage_key, inp.clip_id) == (example["asset_id"], example["storage_key"], example["clip_id"])
    assert inp.storage_key == contracts.source_key(inp.asset_id)
    assert set(example) == {"asset_id", "storage_key", "clip_id"}
    assert contracts.TrackFacesInput.from_json('{"asset_id": "a", "storage_key": "b"}').clip_id == ""  # all clips


def test_translate_input_matches_go(golden):
    example = golden["examples"][contracts.JOB_TRANSLATE_TRANSCRIPT]
    inp = contracts.TranslateTranscriptInput.from_json(json.dumps(example))
    assert (inp.asset_id, inp.language) == (example["asset_id"], "en")
    assert set(example) == {"asset_id", "language"}


def test_message_round_trip():
    msg = contracts.Message("j1", contracts.JOB_TRANSCRIBE_AUDIO, '{"asset_id":"a"}')
    assert contracts.Message.parse(msg.values()) == msg
    assert msg.stream == contracts.STREAM_AI


@pytest.mark.parametrize(
    "values",
    [
        {"type": "t", "payload": "{}"},
        {"job_id": "j", "payload": "{}"},
        {"job_id": "j", "type": "t"},
        {"job_id": "j", "type": "t", "payload": "{"},
    ],
)
def test_message_parse_rejects_bad_input(values):
    with pytest.raises(ValueError):
        contracts.Message.parse(values)
