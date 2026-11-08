"""Asking the language model for one JSON object and checking the answer.
Shared by the jobs that use the model (analyze-transcript, suggest-clips,
translate-transcript).

The answer is checked, not trusted: the JSON is found even with code fences,
reasoning blocks or a sentence around it, and when the job's parser rejects
it the model is asked once more, with the reason.
"""

from __future__ import annotations

import json
import re
from typing import Callable, TypeVar

from .llm import ChatClient, Completion
from .runner import JobContext

MAX_TOKENS = 2048  # answers are small; room for models that think first
REPAIR_PROMPT = "That answer could not be used: {error}. Reply again with only the JSON object described above."

T = TypeVar("T")


class NotConfigured(Exception):
    pass


class InvalidAnswer(ValueError):
    """The model's answer does not have the expected shape."""


def ask(
    client: ChatClient, ctx: JobContext, messages: list[dict], parse: Callable[[str], T], max_tokens: int = MAX_TOKENS
) -> tuple[T, Completion]:
    """Sends messages and parses the answer with parse, which raises
    InvalidAnswer when the answer cannot be used. Asks for a correction once."""
    completion = _complete(client, ctx, messages, max_tokens)
    try:
        return parse(completion.text), completion
    except InvalidAnswer as e:
        ctx.log.warning("the answer could not be used (%s); asking once more", e)
        messages = messages + [
            {"role": "assistant", "content": completion.text},
            {"role": "user", "content": REPAIR_PROMPT.format(error=e)},
        ]
    completion = _complete(client, ctx, messages, max_tokens)
    try:
        return parse(completion.text), completion
    except InvalidAnswer as e:
        raise InvalidAnswer(f"the model's answer could not be used: {e}") from None


def _complete(client: ChatClient, ctx: JobContext, messages: list[dict], max_tokens: int) -> Completion:
    completion = client.complete(messages, max_tokens=max_tokens, sleep=ctx.stop.wait, check=ctx.check)
    if not completion.text.strip() and completion.finish_reason == "length":
        raise InvalidAnswer(f"the model used all {max_tokens} output tokens without answering")
    return completion


_THINKING = re.compile(r"<think(?:ing)?>.*?</think(?:ing)?>", re.DOTALL | re.IGNORECASE)
_FENCE = re.compile(r"^```(?:json)?\s*|\s*```$", re.IGNORECASE)


def extract_json(text: str) -> dict:
    """The JSON object in an answer, tolerating code fences, reasoning blocks
    and a sentence around it."""
    cleaned = _FENCE.sub("", _THINKING.sub("", text or "").strip()).strip()
    if not cleaned:
        raise InvalidAnswer("the answer is empty")
    try:
        data = json.loads(cleaned)
    except ValueError:
        first, last = cleaned.find("{"), cleaned.rfind("}")
        if first < 0 or last <= first:
            raise InvalidAnswer("the answer has no JSON object") from None
        try:
            data = json.loads(cleaned[first : last + 1])
        except ValueError as e:
            raise InvalidAnswer(f"the JSON is not valid ({e.msg} at character {e.pos})") from None
    if not isinstance(data, dict):
        raise InvalidAnswer("the answer is not a JSON object")
    return data


def parse_time(value) -> float | None:
    """Seconds from a number or from "MM:SS" / "H:MM:SS"."""
    if isinstance(value, bool):
        return None
    if isinstance(value, (int, float)):
        return float(value)
    if not isinstance(value, str):
        return None
    parts = value.strip().split(":")
    try:
        numbers = [float(p) for p in parts]
    except ValueError:
        return None
    if len(numbers) == 1:
        return numbers[0]
    if len(numbers) > 3 or any(n < 0 for n in numbers) or any(n >= 60 for n in numbers[1:]):
        return None
    seconds = 0.0
    for n in numbers:
        seconds = seconds * 60 + n
    return seconds


def clock(seconds: float) -> str:
    """MM:SS, or H:MM:SS from one hour."""
    total = int(seconds)
    h, rest = divmod(total, 3600)
    m, s = divmod(rest, 60)
    return f"{h}:{m:02d}:{s:02d}" if h else f"{m:02d}:{s:02d}"


def one_line(value, limit: int) -> str:
    """A string on one line, without surrounding quotes, cut at a word."""
    if not isinstance(value, str):
        return ""
    return shorten(" ".join(value.strip().strip("\"'").split()), limit)


def shorten(text: str, limit: int) -> str:
    if len(text) <= limit:
        return text
    cut = text[: limit - 1].rsplit(" ", 1)[0]
    return cut.rstrip(",;:.") + "…"


def count(n: int | None) -> str:
    return str(n) if n is not None else "?"
