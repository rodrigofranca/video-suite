"""The track-faces job: where the main face is through each suggested clip,
so a vertical export can keep it in frame (docs/adr/0022).

For each clip, frames are decoded from the source (through a signed URL, only
the clip's part) and sampled SAMPLE_FPS times a second; a small face detector
(YuNet, from OpenCV) finds the faces, and the biggest one, preferring the one
seen just before, is the main face. Its positions become a camera path: the
9:16 window stays still while the face is inside its middle part, pans at a
limited speed when it drifts away from it, and cuts when the face leaves the
window and stays out. The path is stored per clip, simplified to the points where it
changes direction, as shares of the source width.
"""

from __future__ import annotations

import logging
import statistics
import threading
from dataclasses import dataclass

from . import contracts
from .config import Config
from .runner import JobContext
from .storage import Storage
from .store import Store

# The detector: OpenCV's YuNet, pinned to a revision of its official repository.
YUNET_REPO = "opencv/face_detection_yunet"
YUNET_REVISION = "3cc26e7f1014a5ee5d74a42acee58bafc9d0a310"
YUNET_FILE = "face_detection_yunet_2023mar.onnx"

SAMPLE_FPS = 5
DETECT_WIDTH = 640  # frames are shrunk to this width before detection
MIN_SCORE = 0.6
MIN_COVERAGE = 0.3  # share of samples with a face needed to follow it

# Camera path, in shares of the 9:16 window width.
DEAD_ZONE = 0.12  # the face can drift this far from the window's centre
MAX_SPEED = 2.0  # window widths per second
CUT_AFTER = 3  # samples the face must stay outside the window to cut to it
SIMPLIFY = 0.004  # tolerance, in shares of the source width


@dataclass(frozen=True)
class Face:
    x: float  # centre, share of the frame width
    area: float  # share of the frame area
    score: float


class FaceTracker:
    def __init__(self, cfg: Config, storage: Storage, store: Store, log: logging.Logger):
        self.cfg, self.storage, self.store, self.log = cfg, storage, store, log
        self._detector = None
        self._lock = threading.Lock()

    def load_model(self):
        """Load YuNet, downloading it on first use (cached in HF_HOME)."""
        with self._lock:
            if self._detector is None:
                import cv2  # heavy import, only when needed
                from huggingface_hub import hf_hub_download

                path = hf_hub_download(YUNET_REPO, YUNET_FILE, revision=YUNET_REVISION)
                self._detector = cv2.FaceDetectorYN.create(path, "", (320, 320), MIN_SCORE, 0.3, 50)
            return self._detector

    def __call__(self, ctx: JobContext, payload: str) -> None:
        inp = contracts.TrackFacesInput.from_json(payload)
        if not contracts.is_id(inp.asset_id) or inp.storage_key != contracts.source_key(inp.asset_id):
            raise ValueError(f"invalid input: asset_id {inp.asset_id!r}, storage_key {inp.storage_key!r}")
        clips = self.store.clip_ranges(inp.asset_id)
        if not clips:
            ctx.log.info("nothing to track: the video has no clips")
            return

        import av  # heavy import, only when needed

        detector = self.load_model()
        url = self.storage.presign_get(inp.storage_key, expires=3600)
        with av.open(url, timeout=30) as container:
            stream = container.streams.video[0]
            stream.thread_type = "AUTO"
            for i, clip in enumerate(clips):
                ctx.check()
                samples, width, height = sample_faces(container, stream, detector, clip["start"], clip["end"], ctx)
                track = build_track(samples, width, height)
                self.store.set_face_track(clip["id"], track)
                ctx.log.info(
                    "clip %d: face in %.0f%% of %d samples; %s",
                    i + 1, track["coverage"] * 100, len(samples),
                    f"{len(track['points'])} camera points" if track["points"] else "the vertical crop stays centred",
                )
                ctx.progress(int((i + 1) * 100 / len(clips)))


def sample_faces(container, stream, detector, start: float, end: float, ctx: JobContext):
    """Decodes [start, end] and returns (time from start, main face or None)
    SAMPLE_FPS times a second, with the frame's width and height."""
    import numpy

    step = 1 / SAMPLE_FPS
    container.seek(max(int(start / stream.time_base), 0), stream=stream, backward=True)
    samples: list[tuple[float, Face | None]] = []
    width = height = 0
    last_slot, previous = -1, None
    for frame in container.decode(stream):
        t = frame.time
        if t is None or t < start - 1e-3:
            continue
        if t > end:
            break
        # One frame per slot of `step` seconds: the first one in it.
        slot = int((t - start) / step + 1e-6)
        if slot <= last_slot:
            continue
        last_slot = slot
        # Phone videos are stored sideways with a rotation to apply when
        # shown; turn the picture as FFmpeg does, so positions match the export.
        turns = round((frame.rotation or 0) / 90) % 4
        width, height = (frame.height, frame.width) if turns % 2 else (frame.width, frame.height)
        dw = DETECT_WIDTH if width > DETECT_WIDTH else width - width % 2
        dh = max(int(height * dw / width) // 2 * 2, 2)
        sw, sh = (dh, dw) if turns % 2 else (dw, dh)
        image = frame.reformat(width=sw, height=sh, format="bgr24").to_ndarray()
        if turns:
            image = numpy.ascontiguousarray(numpy.rot90(image, k=turns))
        detector.setInputSize((dw, dh))
        _, found = detector.detect(image)
        faces = [] if found is None else [
            Face(x=float((f[0] + f[2] / 2) / dw), area=float((f[2] * f[3]) / (dw * dh)), score=float(f[14]))
            for f in found
        ]
        face = main_face(faces, previous)
        previous = face.x if face else previous
        samples.append((round(t - start, 3), face))
        if len(samples) % 25 == 0:
            ctx.check()
    return samples, width, height


def main_face(faces: list[Face], previous: float | None) -> Face | None:
    """The biggest face, preferring one near where the main face was."""
    if not faces:
        return None

    def weight(f: Face) -> float:
        near = previous is None or abs(f.x - previous) < 0.15
        return f.area * (1.0 if near else 0.6)

    return max(faces, key=weight)


def build_track(samples: list[tuple[float, Face | None]], width: int, height: int) -> dict:
    """The stored track: the share of samples with a face and the camera
    path, empty when the face is not found often enough or the source is
    already narrower than 9:16."""
    seen = [(t, f.x) for t, f in samples if f is not None]
    coverage = len(seen) / len(samples) if samples else 0.0
    track = {"coverage": round(coverage, 3), "points": [], "width": width, "height": height}
    window = min(1.0, height * 9 / 16 / width) if width and height else 1.0
    if coverage < MIN_COVERAGE or window >= 0.999:
        return track
    times = [t for t, _ in samples]
    track["points"] = simplify(camera_path(times, fill_gaps(samples), window), SIMPLIFY)
    return track


def fill_gaps(samples: list[tuple[float, Face | None]]) -> list[float]:
    """The face position at every sample: holds the last one seen (the next
    one before the first), then removes one-off jumps with a median of 5."""
    xs: list[float | None] = [f.x if f else None for _, f in samples]
    first = next(x for x in xs if x is not None)
    last = first
    filled = []
    for x in xs:
        last = x if x is not None else last
        filled.append(last)
    return [statistics.median(filled[max(0, i - 2) : i + 3]) for i in range(len(filled))]


def camera_path(times: list[float], xs: list[float], window: float) -> list[dict]:
    """Where to centre the 9:16 window at each sample, as a share of the width."""
    half = window / 2

    def clamp(v: float) -> float:
        return min(max(v, half), 1 - half)

    dead, speed = DEAD_ZONE * window, MAX_SPEED * window
    c = clamp(xs[0])
    points = [{"t": times[0], "x": round(c, 4)}]
    outside = 0
    for i in range(1, len(times)):
        dt = times[i] - times[i - 1]
        diff = xs[i] - c
        if abs(diff) > half:
            # Out of the window: hold still, and cut to the face if it stays
            # there (someone else is speaking), instead of a long pan.
            outside += 1
            if outside >= CUT_AFTER:
                points.append({"t": round(times[i] - 0.04, 3), "x": round(c, 4)})
                c, outside = clamp(xs[i]), 0
        else:
            outside = 0
            if abs(diff) > dead:
                step = min(abs(diff) - dead, speed * dt)
                c = clamp(c + (step if diff > 0 else -step))
        points.append({"t": times[i], "x": round(c, 4)})
    return points


def simplify(points: list[dict], tolerance: float) -> list[dict]:
    """Ramer-Douglas-Peucker: drops points that lie on the line between
    their neighbours, within tolerance."""
    if len(points) < 3:
        return points
    first, last = points[0], points[-1]
    span = last["t"] - first["t"]
    worst, index = -1.0, 0
    for i in range(1, len(points) - 1):
        p = points[i]
        expected = first["x"] + (last["x"] - first["x"]) * ((p["t"] - first["t"]) / span if span else 0)
        if abs(p["x"] - expected) > worst:
            worst, index = abs(p["x"] - expected), i
    if worst <= tolerance:
        return [first, last]
    return simplify(points[: index + 1], tolerance)[:-1] + simplify(points[index:], tolerance)
