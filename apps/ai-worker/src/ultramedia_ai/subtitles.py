"""WebVTT subtitles from Whisper segments."""

from __future__ import annotations

from dataclasses import dataclass, field

MAX_CHARS = 84  # two lines of about 42 characters
MAX_SECONDS = 6.0
LINE_CHARS = 42


@dataclass
class Word:
    start: float
    end: float
    text: str


@dataclass
class Segment:
    start: float
    end: float
    text: str
    words: list[Word] = field(default_factory=list)


@dataclass
class Cue:
    start: float
    end: float
    text: str


def build_cues(segments: list[Segment], max_chars: int = MAX_CHARS, max_seconds: float = MAX_SECONDS) -> list[Cue]:
    """Split segments into readable cues using word timings when available.

    Whisper segments can last up to 30 seconds; subtitles read better in
    chunks of a few seconds and at most two lines.
    """
    cues: list[Cue] = []
    for seg in segments:
        words = [w for w in seg.words if w.text.strip()]
        if not words:
            if seg.text.strip():
                cues.append(Cue(seg.start, seg.end, seg.text.strip()))
            continue
        current: list[Word] = []
        for w in words:
            candidate = _join(current + [w])
            too_long = len(candidate) > max_chars
            too_slow = current and (w.end - current[0].start) > max_seconds
            if current and (too_long or too_slow):
                cues.append(Cue(current[0].start, current[-1].end, _join(current)))
                current = []
            current.append(w)
        if current:
            cues.append(Cue(current[0].start, current[-1].end, _join(current)))
    return cues


def _join(words: list[Word]) -> str:
    # Whisper words carry their leading space (" hello").
    return "".join(w.text for w in words).strip()


def timestamp(seconds: float) -> str:
    """Seconds as a WebVTT timestamp, e.g. 00:01:02.500."""
    ms = max(0, round(seconds * 1000))
    h, ms = divmod(ms, 3_600_000)
    m, ms = divmod(ms, 60_000)
    s, ms = divmod(ms, 1000)
    return f"{h:02d}:{m:02d}:{s:02d}.{ms:03d}"


def wrap(text: str, width: int = LINE_CHARS) -> str:
    """Break text into at most two lines at a word boundary near the middle."""
    if len(text) <= width:
        return text
    middle = len(text) // 2
    left = text.rfind(" ", 0, middle + 1)
    right = text.find(" ", middle)
    candidates = [i for i in (left, right) if i > 0]
    if not candidates:
        return text
    cut = min(candidates, key=lambda i: abs(i - middle))
    return text[:cut].rstrip() + "\n" + text[cut:].lstrip()


def to_webvtt(cues: list[Cue]) -> str:
    lines = ["WEBVTT", ""]
    for i, cue in enumerate(cues, start=1):
        end = max(cue.end, cue.start + 0.1)  # a cue must not end before it starts
        text = cue.text.replace("-->", "→")  # "-->" would break the cue timing line
        lines += [str(i), f"{timestamp(cue.start)} --> {timestamp(end)}", wrap(text), ""]
    return "\n".join(lines)
