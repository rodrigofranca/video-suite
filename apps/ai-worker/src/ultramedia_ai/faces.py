"""The track-faces job: who to keep in frame through each suggested clip, so
a vertical export can follow them (docs/adr/0022, docs/adr/0023).

For each clip, frames are decoded from the source (through a signed URL, only
the clip's part) and sampled SAMPLE_FPS times a second; a small face detector
(YuNet, from OpenCV) finds the faces, which are linked into tracks, one per
person. With one person at a time, the main face is the biggest. When two or
more people are on screen together and the video has sound, the clip is
decoded again at 25 frames a second to cut out each person's face, and an
audio-visual model (Light-ASD, in speakers.py) scores who is speaking; the
camera goes to whoever speaks, and stays with them through pauses.

The chosen face's positions become a camera path: the 9:16 window stays still
while the face is inside its middle part, pans at a limited speed when it
drifts away from it, and cuts when the face leaves the window and stays out
(or, following speakers, as soon as someone else takes the turn). The path is
stored per clip, simplified to the points where it changes direction, as
shares of the source width.
"""

from __future__ import annotations

import logging
import statistics
import threading
from dataclasses import dataclass, field

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

# Tracks: the same face from one sample to the next.
MIN_IOU = 0.3  # overlap with the track's last box to be the same face
MAX_GAP = 3  # samples a face can be missed without ending its track
MIN_HITS = 3  # samples a track needs to count
TOGETHER = 5  # samples with two or more people needed to look for the speaker

# Speakers (scores from the speaker model, above 0 means speaking).
CROP_SCALE = 0.4  # the face crop around the box, as Light-ASD's authors do
CROP_WIDTH = 1280  # frames are shrunk to this width before cropping faces
SMOOTH_FRAMES = 13  # median over the boxes, at 25 frames a second
SWITCH_AFTER = 3  # samples someone else must lead before the camera moves to them
MIN_HOLD = 8  # samples the camera stays with someone before it can move again (no flashes)

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
    box: tuple[float, float, float, float] = (0.0, 0.0, 0.0, 0.0)  # left, top, right, bottom, shares


@dataclass
class Track:
    """One person's face across samples."""

    hits: dict[int, Face] = field(default_factory=dict)  # sample index -> face

    @property
    def first(self) -> int:
        return min(self.hits)

    @property
    def last(self) -> int:
        return max(self.hits)

    def at(self, i: float) -> tuple[float, ...]:
        """Box at a (fractional) sample index, between the nearest hits."""
        keys = sorted(self.hits)
        if i <= keys[0]:
            return self.hits[keys[0]].box
        if i >= keys[-1]:
            return self.hits[keys[-1]].box
        lo = max(k for k in keys if k <= i)
        hi = min(k for k in keys if k >= i)
        if lo == hi:
            return self.hits[lo].box
        w = (i - lo) / (hi - lo)
        return tuple(a + (b - a) * w for a, b in zip(self.hits[lo].box, self.hits[hi].box, strict=True))

    def x_at(self, i: int) -> float:
        left, _, right, _ = self.at(i)
        return (left + right) / 2


class FaceTracker:
    def __init__(self, cfg: Config, storage: Storage, store: Store, log: logging.Logger, speakers=None):
        self.cfg, self.storage, self.store, self.log = cfg, storage, store, log
        self.speakers = speakers  # a speakers.SpeakerModel, or None to follow the biggest face
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
        if (
            not contracts.is_id(inp.asset_id)
            or inp.storage_key != contracts.source_key(inp.asset_id)
            or (inp.clip_id and not contracts.is_id(inp.clip_id))
        ):
            raise ValueError(f"invalid input: asset_id {inp.asset_id!r}, storage_key {inp.storage_key!r}, clip_id {inp.clip_id!r}")
        clips = self.store.clip_ranges(inp.asset_id)
        if inp.clip_id:
            # After an edit: only that clip.
            clips = [c for c in clips if c["id"] == inp.clip_id]
            if not clips:
                ctx.log.info("nothing to track: the clip no longer exists (the clips were probably suggested again)")
                return
        if not clips:
            ctx.log.info("nothing to track: the video has no clips")
            return

        import av  # heavy import, only when needed

        detector = self.load_model()
        url = self.storage.presign_get(inp.storage_key, expires=3600)
        audio_key = contracts.audio_key(inp.asset_id)
        audio_url = self.storage.presign_get(audio_key, expires=3600) if self.storage.exists(audio_key) else None
        with av.open(url, timeout=30) as container:
            stream = container.streams.video[0]
            stream.thread_type = "AUTO"
            for i, clip in enumerate(clips):
                ctx.check()
                start, end = clip["start"], clip["end"]
                samples, width, height = sample_faces(container, stream, detector, start, end, ctx)
                tracks = link_tracks(samples)
                speaking = None
                if together(tracks) >= TOGETHER:
                    speaking = self.follow_speakers(ctx, container, stream, audio_url, samples, tracks, start, end)
                track = build_track(samples, width, height, speaking)
                track["people"] = max_together(tracks)
                track["follows"] = "speaker" if speaking else "face"
                self.store.set_face_track(clip["id"], track)
                ctx.log.info(
                    "clip %d: %s; face in %.0f%% of %d samples; %s",
                    i + 1,
                    f"{track['people']} people, following who speaks" if speaking
                    else f"{track['people']} at a time, following the biggest face",
                    track["coverage"] * 100, len(samples),
                    f"{len(track['points'])} camera points" if track["points"] else "the vertical crop stays centred",
                )
                ctx.progress(int((i + 1) * 100 / len(clips)))

    def follow_speakers(self, ctx, container, stream, audio_url, samples, tracks, start, end):
        """The face to follow at each sample, by who is speaking, or None
        (follow the biggest face) when that cannot be told."""
        if self.speakers is None:
            ctx.log.warning("several people on screen, but the speaker model is not loaded: following the biggest face")
            return None
        if audio_url is None:
            ctx.log.info("several people on screen, but the video has no sound: following the biggest face")
            return None
        from . import speakers

        try:
            pcm = clip_audio(audio_url, start, end)
            mfcc = speakers.mfcc(pcm)
            crops = crop_faces(container, stream, samples, tracks, start, end, ctx)
            scores = {}
            for n, (first_frame, faces) in crops.items():
                frame_scores = self.speakers.scores(mfcc[first_frame * speakers.MFCC_PER_FRAME :], faces)
                scores[n] = per_sample(frame_scores, first_frame, len(samples))
        except Exception as e:
            if ctx.stop.is_set():
                raise
            ctx.log.warning("could not tell who is speaking (%s): following the biggest face", e)
            return None
        return choose_speakers(samples, tracks, scores)


def sample_faces(container, stream, detector, start: float, end: float, ctx: JobContext):
    """Decodes [start, end] and returns (time from start, faces found)
    SAMPLE_FPS times a second, with the frame's width and height."""
    step = 1 / SAMPLE_FPS
    samples: list[tuple[float, list[Face]]] = []
    width = height = 0
    for t, frame in frames_from(container, stream, start, end, step):
        width, height, turns = shown_size(frame)
        dw = DETECT_WIDTH if width > DETECT_WIDTH else width - width % 2
        dh = max(int(height * dw / width) // 2 * 2, 2)
        image = turned(frame, dw, dh, turns, "bgr24")
        detector.setInputSize((dw, dh))
        _, found = detector.detect(image)
        faces = [] if found is None else [
            Face(
                x=float((f[0] + f[2] / 2) / dw), area=float((f[2] * f[3]) / (dw * dh)), score=float(f[14]),
                box=(float(f[0] / dw), float(f[1] / dh), float((f[0] + f[2]) / dw), float((f[1] + f[3]) / dh)),
            )
            for f in found
        ]
        samples.append((round(t, 3), faces))
        if len(samples) % 25 == 0:
            ctx.check()
    return samples, width, height


def frames_from(container, stream, start: float, end: float, step: float):
    """(time from start, frame) for the first frame in each slot of `step`
    seconds of [start, end]."""
    container.seek(max(int(start / stream.time_base), 0), stream=stream, backward=True)
    last_slot = -1
    for frame in container.decode(stream):
        t = frame.time
        if t is None or t < start - 1e-3:
            continue
        if t > end:
            break
        slot = int((t - start) / step + 1e-6)
        if slot <= last_slot:
            continue
        last_slot = slot
        yield t - start, frame


def shown_size(frame) -> tuple[int, int, int]:
    """Width and height as shown, and the quarter turns to get there. Phone
    videos are stored sideways with a rotation to apply when shown; turning
    the picture as FFmpeg does keeps positions matching the export."""
    turns = round((frame.rotation or 0) / 90) % 4
    width, height = (frame.height, frame.width) if turns % 2 else (frame.width, frame.height)
    return width, height, turns


def turned(frame, width: int, height: int, turns: int, fmt: str):
    """The frame as shown, scaled to width x height, as a NumPy array."""
    import numpy

    sw, sh = (height, width) if turns % 2 else (width, height)
    image = frame.reformat(width=sw, height=sh, format=fmt).to_ndarray()
    if turns:
        image = numpy.ascontiguousarray(numpy.rot90(image, k=turns))
    return image


# ---- Tracks ----


def iou(a: tuple[float, ...], b: tuple[float, ...]) -> float:
    w = max(0.0, min(a[2], b[2]) - max(a[0], b[0]))
    h = max(0.0, min(a[3], b[3]) - max(a[1], b[1]))
    inter = w * h
    union = (a[2] - a[0]) * (a[3] - a[1]) + (b[2] - b[0]) * (b[3] - b[1]) - inter
    return inter / union if union > 0 else 0.0


def link_tracks(samples: list[tuple[float, list[Face]]]) -> list[Track]:
    """Links each sample's faces to the faces of the samples before, by
    overlap: one track per person, while they stay in about the same place."""
    tracks: list[Track] = []
    for i, (_, faces) in enumerate(samples):
        live = [tr for tr in tracks if i - tr.last <= MAX_GAP]
        pairs = sorted(
            ((iou(f.box, tr.hits[tr.last].box), fi, ti) for fi, f in enumerate(faces) for ti, tr in enumerate(live)),
            reverse=True,
        )
        used_faces, used_tracks = set(), set()
        for overlap, fi, ti in pairs:
            if overlap < MIN_IOU:
                break
            if fi in used_faces or ti in used_tracks:
                continue
            live[ti].hits[i] = faces[fi]
            used_faces.add(fi)
            used_tracks.add(ti)
        for fi, f in enumerate(faces):
            if fi not in used_faces:
                tracks.append(Track({i: f}))
    return [tr for tr in tracks if len(tr.hits) >= MIN_HITS]


def present(tracks: list[Track], i: int) -> list[Track]:
    return [tr for tr in tracks if tr.first <= i <= tr.last]


def max_together(tracks: list[Track]) -> int:
    ends = {i for tr in tracks for i in (tr.first, tr.last)}
    return max((len(present(tracks, i)) for i in ends), default=0)


def together(tracks: list[Track]) -> int:
    """Samples with two or more people on screen."""
    if len(tracks) < 2:
        return 0
    span = range(min(tr.first for tr in tracks), max(tr.last for tr in tracks) + 1)
    return sum(1 for i in span if len(present(tracks, i)) >= 2)


# ---- Speakers ----


def clip_audio(url: str, start: float, end: float):
    """The clip's sound as 16 kHz mono int16 samples, from start."""
    import av
    import numpy

    resampler = av.AudioResampler(format="s16", layout="mono", rate=16000)
    chunks, first = [], None
    with av.open(url, timeout=30) as container:
        stream = container.streams.audio[0]
        container.seek(max(int(start / stream.time_base), 0), stream=stream, backward=True)
        for frame in container.decode(stream):
            if frame.time is not None and frame.time > end:
                break
            for out in resampler.resample(frame):
                if out.time is None:
                    continue
                if first is None:
                    first = out.time
                chunks.append(out.to_ndarray().reshape(-1))
    if not chunks:
        raise ValueError("the clip has no sound")
    pcm = numpy.concatenate(chunks)
    skip = round((start - first) * 16000)
    if skip > 0:
        pcm = pcm[skip:]
    elif skip < 0:
        pcm = numpy.concatenate([numpy.zeros(-skip, numpy.int16), pcm])
    return pcm[: round((end - start) * 16000)]


def crop_faces(container, stream, samples, tracks, start: float, end: float, ctx: JobContext):
    """Each track's face, 112x112 gray, 25 times a second, cut out as
    Light-ASD's authors do (the lower part of a box 40% wider than the face,
    smoothed). Returns {track index: (first frame, faces [T, 112, 112])}."""
    import cv2
    import numpy

    from .speakers import FACE_SIZE, FPS

    frame_count = int((end - start) * FPS)
    spans = {}
    for n, tr in enumerate(tracks):
        first = round(samples[tr.first][0] * FPS)
        last = min(round(samples[tr.last][0] * FPS) + FPS // SAMPLE_FPS, frame_count)
        if last - first < 10:
            continue
        # Box per frame, between the samples, then a median to steady it.
        boxes = numpy.array([tr.at((k / FPS) * SAMPLE_FPS) for k in range(first, last)])
        cx, cy = (boxes[:, 0] + boxes[:, 2]) / 2, (boxes[:, 1] + boxes[:, 3]) / 2
        bw, bh = boxes[:, 2] - boxes[:, 0], boxes[:, 3] - boxes[:, 1]
        spans[n] = (first, last, *(steady(v) for v in (cx, cy, bw, bh)))
    if not spans:
        return {}
    crops = {n: [None] * (last - first) for n, (first, last, *_) in spans.items()}
    for t, frame in frames_from(container, stream, start, end, 1 / FPS):
        k = int(t * FPS + 1e-6)
        width, height, turns = shown_size(frame)
        gw = min(width, CROP_WIDTH)
        gh = max(int(height * gw / width), 2)
        gray = None
        for n, (first, last, cx, cy, bw, bh) in spans.items():
            if not first <= k < last:
                continue
            if gray is None:
                gray = turned(frame, gw, gh, turns, "gray")
            j = k - first
            size = max(bw[j] * gw, bh[j] * gh) / 2
            crops[n][j] = cut(gray, cx[j] * gw, cy[j] * gh, size, cv2, numpy, FACE_SIZE)
        if k % 125 == 0:
            ctx.check()
    out = {}
    for n, faces in crops.items():
        # Frames the decoder skipped repeat the one before, to stay in step with the sound.
        filled, last_face = [], next((f for f in faces if f is not None), None)
        if last_face is None:
            continue
        for f in faces:
            last_face = f if f is not None else last_face
            filled.append(last_face)
        out[n] = (spans[n][0], numpy.stack(filled))
    return out


def steady(values):
    """Running median over SMOOTH_FRAMES, repeating the ends."""
    import numpy

    half = SMOOTH_FRAMES // 2
    padded = numpy.pad(values, half, mode="edge")
    windows = numpy.lib.stride_tricks.sliding_window_view(padded, SMOOTH_FRAMES)
    return numpy.median(windows, axis=1)


def cut(gray, cx: float, cy: float, size: float, cv2, numpy, face_size: int):
    """Light-ASD's face crop: a square of 2.8 x size from size above the face
    centre, padded with gray, shrunk to twice face_size, the middle kept."""
    pad = int(size * (1 + 2 * CROP_SCALE))
    image = numpy.pad(gray, pad, mode="constant", constant_values=110)
    top, left = int(cy + pad - size), int(cx + pad - size * (1 + CROP_SCALE))
    bottom, right = int(cy + pad + size * (1 + 2 * CROP_SCALE)), int(cx + pad + size * (1 + CROP_SCALE))
    face = image[max(top, 0) : bottom, max(left, 0) : right]
    if face.size == 0:
        return numpy.full((face_size, face_size), 110, numpy.uint8)
    face = cv2.resize(face, (face_size * 2, face_size * 2))
    q = face_size // 2
    return face[q : q + face_size, q : q + face_size]


def per_sample(frame_scores, first_frame: int, count: int) -> dict[int, float]:
    """A track's score per sample: the mean over the sample's frames, after
    a mean over 5 frames, as the authors smooth them."""
    import numpy

    from .speakers import FPS

    if len(frame_scores) == 0:
        return {}
    smooth = numpy.convolve(numpy.pad(frame_scores, 2, mode="edge"), numpy.ones(5) / 5, mode="valid")
    per = FPS // SAMPLE_FPS
    out = {}
    for i in range(count):
        a, b = i * per - first_frame, (i + 1) * per - first_frame
        if b <= 0 or a >= len(smooth):
            continue
        out[i] = float(smooth[max(a, 0) : min(b, len(smooth))].mean())
    return out


def choose_speakers(samples, tracks: list[Track], scores: dict[int, dict[int, float]]) -> list[float | None]:
    """Where the face to follow is at each sample: whoever speaks (scores
    above 0, the highest), held through pauses; someone else must lead for
    SWITCH_AFTER samples before the camera moves to them, and not before
    MIN_HOLD samples on the same person. Without anyone speaking yet, the
    biggest face."""
    current, pending, count, held = None, None, 0, 0
    xs: list[float | None] = []
    for i in range(len(samples)):
        here = [n for n, tr in enumerate(tracks) if tr.first <= i <= tr.last]
        if not here:
            xs.append(None)
            continue
        talking = [(scores[n][i], n) for n in here if n in scores and i in scores[n] and scores[n][i] > 0]
        if talking:
            candidate = max(talking)[1]
        elif current in here:
            candidate = current
        else:
            candidate = max(here, key=lambda n: area(tracks[n], i))
        held += 1
        if current not in here:
            current, count, held = candidate, 0, 0
        elif candidate != current:
            count = count + 1 if candidate == pending else 1
            pending = candidate
            if count >= SWITCH_AFTER and held >= MIN_HOLD:
                current, count, held = candidate, 0, 0
        else:
            count = 0
        xs.append(tracks[current].x_at(i))
    return xs


def area(track: Track, i: int) -> float:
    left, top, right, bottom = track.at(i)
    return (right - left) * (bottom - top)


# ---- Camera path ----


def main_face(faces: list[Face], previous: float | None) -> Face | None:
    """The biggest face, preferring one near where the main face was."""
    if not faces:
        return None

    def weight(f: Face) -> float:
        near = previous is None or abs(f.x - previous) < 0.15
        return f.area * (1.0 if near else 0.6)

    return max(faces, key=weight)


def biggest_faces(samples: list[tuple[float, list[Face]]]) -> list[float | None]:
    xs, previous = [], None
    for _, faces in samples:
        face = main_face(faces, previous)
        previous = face.x if face else previous
        xs.append(face.x if face else None)
    return xs


def build_track(samples: list[tuple[float, list[Face]]], width: int, height: int,
                speaking: list[float | None] | None = None) -> dict:
    """The stored track: the share of samples with a face and the camera
    path, following `speaking` (positions per sample) when given and the
    biggest face otherwise; empty when faces are not found often enough or
    the source is already narrower than 9:16."""
    seen = sum(1 for _, faces in samples if faces)
    coverage = seen / len(samples) if samples else 0.0
    track = {"coverage": round(coverage, 3), "points": [], "width": width, "height": height}
    window = min(1.0, height * 9 / 16 / width) if width and height else 1.0
    if coverage < MIN_COVERAGE or window >= 0.999:
        return track
    times = [t for t, _ in samples]
    xs = speaking if speaking is not None else biggest_faces(samples)
    # A change of speaker was already held for SWITCH_AFTER samples: cut at once.
    cut_after = 1 if speaking is not None else CUT_AFTER
    track["points"] = simplify(camera_path(times, fill_gaps(xs), window, cut_after), SIMPLIFY)
    return track


def fill_gaps(xs: list[float | None]) -> list[float]:
    """The face position at every sample: holds the last one seen (the next
    one before the first), then removes one-off jumps with a median of 5."""
    first = next(x for x in xs if x is not None)
    last = first
    filled = []
    for x in xs:
        last = x if x is not None else last
        filled.append(last)
    return [statistics.median(filled[max(0, i - 2) : i + 3]) for i in range(len(filled))]


def camera_path(times: list[float], xs: list[float], window: float, cut_after: int = CUT_AFTER) -> list[dict]:
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
            if outside >= cut_after:
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
