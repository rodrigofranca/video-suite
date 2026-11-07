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
from ultramedia_ai.faces import (
    Face,
    FaceTracker,
    build_track,
    camera_path,
    fill_gaps,
    main_face,
    simplify,
)
from ultramedia_ai.runner import JobContext

FACE = Path(__file__).parent / "fixtures" / "face.jpg"
WINDOW = 720 * 9 / 16 / 1280  # share of a 1280x720 frame a 9:16 crop shows


def samples(xs):
    return [(round(i * 0.2, 3), Face(x, 0.05, 0.9) if x is not None else None) for i, x in enumerate(xs)]


# ---- Camera path ----


def test_main_face_prefers_the_biggest_near_the_last_one():
    big_far, small_near = Face(0.8, 0.10, 0.9), Face(0.3, 0.07, 0.9)
    assert main_face([big_far, small_near], previous=None) == big_far
    assert main_face([big_far, small_near], previous=0.32) == small_near  # 0.10 * 0.6 < 0.07
    assert main_face([], previous=0.5) is None


def test_fill_gaps_holds_positions_and_drops_one_off_jumps():
    xs = fill_gaps(samples([None, 0.4, None, 0.41, 0.9, 0.42, 0.43]))
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


def test_build_track():
    assert build_track(samples([None] * 8 + [0.5, 0.5]), 1280, 720)["points"] == []  # face in 20% of samples
    portrait = build_track(samples([0.5] * 10), 720, 1280)
    assert portrait["coverage"] == 1 and portrait["points"] == []  # already narrower than 9:16
    track = build_track(samples([0.3] * 10), 1280, 720)
    assert (track["coverage"], track["width"], track["height"]) == (1, 1280, 720)
    assert [p["x"] for p in track["points"]] == [0.3, 0.3]


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


def test_rejects_foreign_keys_and_skips_assets_without_clips(tracker, asset, store):
    ctx = JobContext(
        job=type("J", (), {"id": asset["jobs"][contracts.JOB_TRACK_FACES]})(), log=logging.getLogger("test"),
        deadline=time.monotonic() + 60, stop=threading.Event(), timeout_seconds=60,
    )
    with pytest.raises(ValueError, match="invalid input"):
        tracker(ctx, f'{{"asset_id": "{asset["id"]}", "storage_key": "assets/other/source/input.mp4"}}')
    tracker(ctx, f'{{"asset_id": "{asset["id"]}", "storage_key": "{contracts.source_key(asset["id"])}"}}')
    assert store.clip_ranges(asset["id"]) == []
