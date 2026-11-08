"""Face tracking: the camera path from face positions, and the job end to end
with a real face moving across a video (FFmpeg, the YuNet model, storage and
the database)."""

import logging
import shutil
import subprocess
import threading
import time
from itertools import pairwise
from pathlib import Path

import pytest

from ultramedia_ai import contracts
from ultramedia_ai import faces as F
from ultramedia_ai.faces import (
    Face,
    FaceTracker,
    Track,
    build_track,
    camera_path,
    choose_speakers,
    fill_gaps,
    link_tracks,
    main_face,
    per_sample,
    simplify,
)
from ultramedia_ai.runner import JobContext
from ultramedia_ai.speakers import SpeakerModel

FACE = Path(__file__).parent / "fixtures" / "face.jpg"
WINDOW = 720 * 9 / 16 / 1280  # share of a 1280x720 frame a 9:16 crop shows


def face(x, size=0.2, y=0.5):
    """A square-ish face centred at x, size wide, in a 16:9 frame."""
    h = size * 16 / 9
    return Face(x, size * h, 0.9, (x - size / 2, y - h / 2, x + size / 2, y + h / 2))


def samples(*people):
    """Samples 0.2 s apart; each person is a list of x positions (None: not seen)."""
    count = max(len(xs) for xs in people)
    return [
        (round(i * 0.2, 3), [face(xs[i]) for xs in people if i < len(xs) and xs[i] is not None])
        for i in range(count)
    ]


# ---- Camera path ----


def test_main_face_prefers_the_biggest_near_the_last_one():
    big_far, small_near = Face(0.8, 0.10, 0.9), Face(0.3, 0.07, 0.9)  # x, area, score
    assert main_face([big_far, small_near], previous=None) == big_far
    assert main_face([big_far, small_near], previous=0.32) == small_near  # 0.10 * 0.6 < 0.07
    assert main_face([], previous=0.5) is None


def test_fill_gaps_holds_positions_and_drops_one_off_jumps():
    xs = fill_gaps([None, 0.4, None, 0.41, 0.9, 0.42, 0.43])
    assert xs[0] == 0.4 and xs[2] == 0.4
    assert max(xs) < 0.5  # the single 0.9 is noise


def test_a_still_face_keeps_the_window_still():
    times = [i * 0.2 for i in range(20)]
    path = camera_path(times, [0.5 + (0.01 if i % 2 else -0.01) for i in range(20)], WINDOW)
    assert len({p["x"] for p in path}) == 1  # inside the dead zone: no movement
    assert len(simplify(path, 0.004)) == 2


def test_the_window_pans_at_a_limited_speed_and_stays_in_the_frame():
    times = [i * 0.2 for i in range(30)]
    xs = [0.2 + 0.02 * i for i in range(30)]  # walks right, 0.1 of the width a second
    path = camera_path(times, xs, WINDOW)
    centres = [p["x"] for p in path]
    assert centres == sorted(centres) and centres[-1] > centres[0] + 0.4
    steps = [b - a for a, b in pairwise(centres)]
    assert max(steps) <= 2.0 * WINDOW * 0.2 + 1e-9
    assert all(WINDOW / 2 - 1e-9 <= c <= 1 - WINDOW / 2 + 1e-9 for c in centres)
    assert abs(centres[-1] - xs[-1]) <= 0.12 * WINDOW + 1e-3  # it catches up, within the dead zone (positions have 4 decimals)


def test_a_face_that_jumps_away_and_stays_gets_a_cut():
    times = [i * 0.2 for i in range(12)]
    xs = [0.25] * 5 + [0.8] * 7  # someone else is speaking now
    path = camera_path(times, xs, WINDOW)
    assert path[-1]["x"] == pytest.approx(0.8)
    jumps = [(a, b) for a, b in pairwise(path) if abs(b["x"] - a["x"]) > 0.3]
    assert len(jumps) == 1 and jumps[0][1]["t"] - jumps[0][0]["t"] <= 0.05  # a cut, not a pan


def test_a_cut_can_be_immediate():
    times = [i * 0.2 for i in range(6)]
    path = camera_path(times, [0.25] * 3 + [0.8] * 3, WINDOW, cut_after=1)
    assert next(p["t"] for p in path if p["x"] == pytest.approx(0.8)) == pytest.approx(0.6)


def test_build_track():
    assert build_track(samples([None] * 8 + [0.5, 0.5]), 1280, 720)["points"] == []  # face in 20% of samples
    portrait = build_track(samples([0.5] * 10), 720, 1280)
    assert portrait["coverage"] == 1 and portrait["points"] == []  # already narrower than 9:16
    track = build_track(samples([0.3] * 10), 1280, 720)
    assert (track["coverage"], track["width"], track["height"]) == (1, 1280, 720)
    assert [p["x"] for p in track["points"]] == [0.3, 0.3]


# ---- Tracks and speakers ----


def test_tracks_follow_each_person_through_short_misses():
    left = [0.25] * 10
    right = [0.75, 0.75, None, None, 0.76, 0.76, 0.77, 0.77, 0.77, 0.77]  # missed twice
    tracks = sorted(link_tracks(samples(left, right)), key=lambda tr: tr.x_at(0))
    assert [len(tr.hits) for tr in tracks] == [10, 8]
    assert tracks[0].x_at(5) == pytest.approx(0.25)
    assert 0.75 <= tracks[1].x_at(2) <= 0.76  # between the samples it was seen in
    assert F.max_together(tracks) == 2 and F.together(tracks) == 10


def test_a_face_seen_once_is_not_a_track():
    assert len(link_tracks(samples([0.5] * 6, [None, None, 0.9]))) == 1


def two_people(n=30):
    s = samples([0.25] * n, [0.75] * n)
    tracks = sorted(link_tracks(s), key=lambda tr: tr.x_at(0))
    return s, tracks


def talk(n, left_speaks):
    """Scores for tracks 0 (left) and 1 (right): left_speaks(i) says who talks."""
    return {
        0: {i: 3.0 if left_speaks(i) else -2.0 for i in range(n)},
        1: {i: -2.0 if left_speaks(i) else 3.0 for i in range(n)},
    }


def test_the_camera_goes_to_whoever_speaks_after_a_steady_lead():
    s, tracks = two_people()
    xs = choose_speakers(s, tracks, talk(30, lambda i: i < 15))
    assert xs[:15] == [pytest.approx(0.25)] * 15
    assert xs.index(pytest.approx(0.75)) == 15 + F.SWITCH_AFTER - 1
    assert xs[-1] == pytest.approx(0.75)


def test_a_short_interjection_does_not_move_the_camera():
    s, tracks = two_people()
    xs = choose_speakers(s, tracks, talk(30, lambda i: not 10 <= i < 10 + F.SWITCH_AFTER - 1))
    assert xs == [pytest.approx(0.25)] * 30


def test_the_camera_holds_a_shot_before_moving_again():
    s, tracks = two_people()
    # Left until 10, right for 3 samples, then left again.
    xs = choose_speakers(s, tracks, talk(30, lambda i: not 10 <= i < 13))
    moved = xs.index(pytest.approx(0.75))
    back = moved + xs[moved:].index(pytest.approx(0.25))
    assert back - moved == F.MIN_HOLD


def test_pauses_keep_the_last_speaker_and_nobody_speaking_means_the_biggest_face():
    s = [(round(i * 0.2, 3), [face(0.25, 0.1), face(0.75, 0.25)]) for i in range(20)]
    tracks = sorted(link_tracks(s), key=lambda tr: tr.x_at(0))
    silent = {0: dict.fromkeys(range(20), -1.0), 1: dict.fromkeys(range(20), -1.0)}
    assert choose_speakers(s, tracks, silent) == [pytest.approx(0.75)] * 20  # the bigger one
    left_then_quiet = {0: {i: 2.0 if i < 5 else -1.0 for i in range(20)}, 1: dict.fromkeys(range(20), -1.0)}
    assert choose_speakers(s, tracks, left_then_quiet) == [pytest.approx(0.25)] * 20


def test_scores_per_sample_average_each_sample_s_frames():
    frames = [1.0] * 10 + [-1.0] * 10  # 20 frames from frame 5 (the second sample)
    out = per_sample(frames, first_frame=5, count=6)
    assert 0 not in out and 5 not in out
    assert out[1] == pytest.approx(1.0) and out[4] == pytest.approx(-1.0)
    assert out[2] > 0 > out[3]


def test_track_boxes_are_interpolated_between_hits():
    tr = Track({0: face(0.2), 2: face(0.4)})
    assert tr.x_at(1) == pytest.approx(0.3)
    assert tr.x_at(5) == pytest.approx(0.4)


# ---- The job ----


def moving_face_video(path):
    """6 s of 1280x720 gray with the face moving from left to right."""
    if not shutil.which("ffmpeg"):
        pytest.skip("ffmpeg not installed")
    subprocess.run(
        ["ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i", "color=c=0x606060:s=1280x720:r=25:d=6",
         "-loop", "1", "-i", str(FACE), "-filter_complex", "[0][1]overlay=x='100+t*130':y=240:shortest=1",
         "-c:v", "libx264", "-pix_fmt", "yuv420p", "-t", "6", str(path)],
        check=True,
    )


@pytest.fixture
def tracker(cfg, storage, store):
    t = FaceTracker(cfg, storage, store, logging.getLogger("test"))
    try:
        t.load_model()
    except Exception as e:  # noqa: BLE001
        pytest.skip(f"face model not available: {e}")
    return t


def test_tracks_a_face_moving_across_the_clip(cfg, storage, store, asset, tracker, tmp_path, caplog):
    caplog.set_level(logging.INFO)
    video = tmp_path / "moving.mp4"
    moving_face_video(video)
    key = contracts.source_key(asset["id"])
    storage.upload_file(key, str(video), "video/mp4")
    try:
        store.replace_clips(asset["id"], asset["jobs"][contracts.JOB_SUGGEST_CLIPS], [
            {"start": 0.0, "end": 6.0, "title": "Whole", "reason": "", "score": 8, "rank": 1},
            {"start": 3.0, "end": 5.0, "title": "Second half", "reason": "", "score": 6, "rank": 2},
        ], "test")
        ctx = JobContext(
            job=type("J", (), {"id": asset["jobs"][contracts.JOB_TRACK_FACES]})(), log=logging.getLogger("test"),
            deadline=time.monotonic() + 120, stop=threading.Event(), timeout_seconds=120,
        )
        tracker(ctx, f'{{"asset_id": "{asset["id"]}", "storage_key": "{key}"}}')

        whole, second = (store.get_face_track(c["id"]) for c in store.clip_ranges(asset["id"]))
        assert whole["coverage"] > 0.8 and (whole["width"], whole["height"]) == (1280, 720)
        assert "30 samples" in caplog.text  # 6 s at 5 samples a second, not more
        xs = [p["x"] for p in whole["points"]]
        assert xs == sorted(xs), "the window follows the face to the right"
        assert xs[0] < 0.3 and xs[-1] > 0.65
        assert whole["points"][0]["t"] == 0 and 5.5 <= whole["points"][-1]["t"] <= 6.0
        # The second clip's times start at its own start, where the face is already on the right.
        assert second["points"][0]["t"] == 0 and second["points"][0]["x"] > 0.4
    finally:
        storage.delete(key)


def test_rotated_phone_videos_are_tracked_as_shown(cfg, storage, store, asset, tracker, tmp_path):
    """Phone videos are stored turned, with a rotation to apply when shown:
    the face must be looked for, and placed, in the picture as shown."""
    video = tmp_path / "moving.mp4"
    moving_face_video(video)
    key = contracts.source_key(asset["id"])
    store.replace_clips(asset["id"], asset["jobs"][contracts.JOB_SUGGEST_CLIPS], [
        {"start": 0.0, "end": 6.0, "title": "Whole", "reason": "", "score": 8, "rank": 1},
    ], "test")
    ctx = JobContext(
        job=type("J", (), {"id": asset["jobs"][contracts.JOB_TRACK_FACES]})(), log=logging.getLogger("test"),
        deadline=time.monotonic() + 120, stop=threading.Event(), timeout_seconds=120,
    )
    # Stored upside down or turned a quarter clockwise (720x1280), shown as the original.
    try:
        for rotation, turn in (("180", "hflip,vflip"), ("90", "transpose=clock")):
            stored, rotated = tmp_path / f"stored{rotation}.mp4", tmp_path / f"rotated{rotation}.mp4"
            subprocess.run(["ffmpeg", "-v", "error", "-y", "-i", str(video), "-vf", turn,
                            "-c:v", "libx264", "-pix_fmt", "yuv420p", str(stored)], check=True)
            done = subprocess.run(["ffmpeg", "-v", "error", "-y", "-display_rotation", rotation, "-i", str(stored),
                                   "-c", "copy", str(rotated)], capture_output=True, check=False)
            if done.returncode:
                pytest.skip(f"this FFmpeg cannot set a rotation: {done.stderr!r}")
            storage.upload_file(key, str(rotated), "video/mp4")
            tracker(ctx, f'{{"asset_id": "{asset["id"]}", "storage_key": "{key}"}}')
            (track,) = (store.get_face_track(c["id"]) for c in store.clip_ranges(asset["id"]))
            xs = [p["x"] for p in track["points"]]
            assert (track["width"], track["height"]) == (1280, 720), rotation
            assert track["coverage"] > 0.8, rotation
            assert xs == sorted(xs) and xs[0] < 0.3 and xs[-1] > 0.65, (rotation, xs)
    finally:
        storage.delete(key)


@pytest.fixture
def speaking_tracker(cfg, storage, store, tracker):
    from huggingface_hub import constants

    model = SpeakerModel(constants.HF_HOME)
    try:
        model.load()
    except Exception as e:  # noqa: BLE001
        pytest.skip(f"speaker model not available: {e}")
    t = FaceTracker(cfg, storage, store, logging.getLogger("test"), model)
    t._detector = tracker.load_model()
    return t


def test_two_people_in_a_clip_with_sound_go_through_the_speaker_model(
    storage, store, asset, speaking_tracker, tmp_path, caplog
):
    """Two still faces and a tone: the clip is decoded twice, both faces are
    scored, nobody is speaking, so the camera stays on the bigger face."""
    caplog.set_level(logging.INFO)
    video, audio = tmp_path / "two.mp4", tmp_path / "audio.flac"
    if not shutil.which("ffmpeg"):
        pytest.skip("ffmpeg not installed")
    subprocess.run(
        ["ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i", "color=c=0x606060:s=1280x720:r=25:d=4",
         "-loop", "1", "-i", str(FACE), "-filter_complex",
         "[1]split[a][b];[a]scale=180:180[s];[b]scale=300:300[l];[0][s]overlay=150:270[t];[t][l]overlay=850:210:shortest=1",
         "-c:v", "libx264", "-pix_fmt", "yuv420p", "-t", "4", str(video)],
        check=True,
    )
    subprocess.run(["ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i", "sine=frequency=300:duration=4",
                    "-ac", "1", "-ar", "16000", str(audio)], check=True)
    key, akey = contracts.source_key(asset["id"]), contracts.audio_key(asset["id"])
    storage.upload_file(key, str(video), "video/mp4")
    storage.upload_file(akey, str(audio), "audio/flac")
    try:
        store.replace_clips(asset["id"], asset["jobs"][contracts.JOB_SUGGEST_CLIPS], [
            {"start": 0.5, "end": 3.5, "title": "Both", "reason": "", "score": 8, "rank": 1},
        ], "test")
        ctx = JobContext(
            job=type("J", (), {"id": asset["jobs"][contracts.JOB_TRACK_FACES]})(), log=logging.getLogger("test"),
            deadline=time.monotonic() + 120, stop=threading.Event(), timeout_seconds=120,
        )
        speaking_tracker(ctx, f'{{"asset_id": "{asset["id"]}", "storage_key": "{key}"}}')
        (track,) = (store.get_face_track(c["id"]) for c in store.clip_ranges(asset["id"]))
        assert "2 people, following who speaks" in caplog.text
        assert (track["follows"], track["people"]) == ("speaker", 2)
        bigger = (850 + 300 / 2) / 1280  # the centre of the bigger face
        assert track["points"] and all(p["x"] == pytest.approx(bigger, abs=0.02) for p in track["points"])
    finally:
        storage.delete(key)
        storage.delete(akey)


def test_rejects_foreign_keys_and_skips_assets_without_clips(tracker, asset, store):
    ctx = JobContext(
        job=type("J", (), {"id": asset["jobs"][contracts.JOB_TRACK_FACES]})(), log=logging.getLogger("test"),
        deadline=time.monotonic() + 60, stop=threading.Event(), timeout_seconds=60,
    )
    with pytest.raises(ValueError, match="invalid input"):
        tracker(ctx, f'{{"asset_id": "{asset["id"]}", "storage_key": "assets/other/source/input.mp4"}}')
    tracker(ctx, f'{{"asset_id": "{asset["id"]}", "storage_key": "{contracts.source_key(asset["id"])}"}}')
    assert store.clip_ranges(asset["id"]) == []
