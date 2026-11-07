"""Chat completions through the language model gateway (LiteLLM,
docs/adr/0018). The gateway speaks the OpenAI API, so this client knows
nothing about the provider behind it: the provider and its key are set in the
gateway, and the worker only names a model alias such as "ultramedia-text".

Uses the standard library only: one POST per call does not need more.
"""

from __future__ import annotations

import json
import re
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from typing import Callable


class LLMError(Exception):
    """The gateway could not give an answer. The message never holds the key."""


# Statuses worth another try: timeouts, rate limits and gateway or provider outages.
RETRY_STATUSES = {408, 409, 429, 500, 502, 503, 504}
MAX_RETRY_AFTER = 30.0  # seconds; a longer Retry-After fails the job instead


@dataclass(frozen=True)
class Completion:
    text: str
    model: str  # the model that answered, as the gateway reports it
    finish_reason: str
    prompt_tokens: int | None = None
    completion_tokens: int | None = None


class ChatClient:
    def __init__(
        self,
        base_url: str,
        api_key: str,
        model: str,
        timeout: float = 120,
        attempts: int = 3,
        backoff: float = 2.0,
    ):
        if not base_url:
            raise ValueError("base_url is required")
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.model = model
        self.timeout = timeout
        self.attempts = max(1, attempts)
        self.backoff = backoff

    @property
    def host(self) -> str:
        """Where the gateway is, for logs (no path, no credentials)."""
        parts = urllib.parse.urlsplit(self.base_url)
        return parts.hostname + (f":{parts.port}" if parts.port else "") if parts.hostname else self.base_url

    def complete(
        self,
        messages: list[dict],
        *,
        max_tokens: int,
        temperature: float = 0.2,
        sleep: Callable[[float], object] = time.sleep,
        check: Callable[[], None] | None = None,
    ) -> Completion:
        """Send a chat completion request, retrying transient failures.

        sleep waits between attempts (a job passes its stop event's wait) and
        check runs before each attempt, so a shutdown or timeout stops retries.
        """
        body = {"model": self.model, "messages": messages, "max_tokens": max_tokens, "temperature": temperature}
        error: LLMError | None = None
        for attempt in range(1, self.attempts + 1):
            if check is not None:
                check()
            wait = self.backoff * 2 ** (attempt - 1)
            try:
                status, headers, raw = self._post("/chat/completions", body)
            except (urllib.error.URLError, TimeoutError, OSError) as e:
                error = LLMError(f"could not reach the language model gateway at {self.host}: {_reason(e)}")
            else:
                if status == 200:
                    return _completion(raw, headers, self.model)
                message = _error_message(raw)
                error = LLMError(f"language model gateway answered {status}: {message}{_hint(status, message)}")
                if status not in RETRY_STATUSES or _is_auth_error(message):
                    raise error
                retry_after = _retry_after(headers.get("Retry-After"))
                if retry_after is not None:
                    if retry_after > MAX_RETRY_AFTER:
                        raise error
                    wait = max(wait, retry_after)
            if attempt < self.attempts:
                sleep(wait)
        assert error is not None
        raise error

    def models(self) -> list[str]:
        """The model names the gateway offers (GET /models)."""
        status, _, raw = self._get("/models", timeout=5)
        if status != 200:
            raise LLMError(f"language model gateway answered {status}: {_error_message(raw)}")
        data = json.loads(raw)
        return [m.get("id", "") for m in data.get("data", []) if isinstance(m, dict)]

    # ---- HTTP ----

    def _headers(self) -> dict[str, str]:
        headers = {"Content-Type": "application/json", "Accept": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        return headers

    def _post(self, path: str, body: dict) -> tuple[int, dict, bytes]:
        req = urllib.request.Request(
            self.base_url + path, data=json.dumps(body).encode(), headers=self._headers(), method="POST"
        )
        return self._send(req, self.timeout)

    def _get(self, path: str, timeout: float) -> tuple[int, dict, bytes]:
        return self._send(urllib.request.Request(self.base_url + path, headers=self._headers()), timeout)

    @staticmethod
    def _send(req: urllib.request.Request, timeout: float) -> tuple[int, dict, bytes]:
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                return resp.status, dict(resp.headers), resp.read()
        except urllib.error.HTTPError as e:  # 4xx and 5xx still carry a body worth reading
            with e:
                return e.code, dict(e.headers or {}), e.read()


def _completion(raw: bytes, headers: dict, fallback_model: str) -> Completion:
    try:
        data = json.loads(raw)
        choice = data["choices"][0]
        content = choice["message"].get("content")
    except (ValueError, KeyError, IndexError, TypeError, AttributeError) as e:
        raise LLMError(f"unexpected answer from the language model gateway: {e.__class__.__name__}") from None
    if isinstance(content, list):  # some providers return content parts
        content = "".join(p.get("text", "") for p in content if isinstance(p, dict))
    usage = data.get("usage") or {}
    return Completion(
        text=content or "",
        model=_answering_model(headers) or str(data.get("model") or fallback_model),
        finish_reason=str(choice.get("finish_reason") or ""),
        prompt_tokens=usage.get("prompt_tokens"),
        completion_tokens=usage.get("completion_tokens"),
    )


def _answering_model(headers: dict) -> str:
    """The provider's model behind the alias. LiteLLM answers with the alias
    in the body and names the real model in a header, with its own provider
    prefix ("openai/deepseek-v4-flash")."""
    name = next((v for k, v in headers.items() if k.lower() == "x-litellm-model-name"), "")
    return name.split("/", 1)[1] if "/" in name else name


# Anything that looks like a key, in case a provider echoes one back.
_SECRET = re.compile(r"(sk-|Bearer\s+)[A-Za-z0-9._\-]{4,}|\b[A-Za-z0-9_\-]{32,}\b")


def _error_message(raw: bytes) -> str:
    """The gateway's error message, shortened and without anything key-like."""
    text = raw.decode("utf-8", errors="replace").strip()
    try:
        data = json.loads(text)
        err = data.get("error", data) if isinstance(data, dict) else data
        text = str(err.get("message") or err) if isinstance(err, dict) else str(err)
    except ValueError:
        pass
    text = _SECRET.sub("[redacted]", " ".join(text.split()))
    return (text[:300] + "…") if len(text) > 300 else (text or "no details")


def _is_auth_error(message: str) -> bool:
    # LiteLLM reports a provider's missing or rejected key as an
    # AuthenticationError, sometimes with status 500. Retrying cannot fix it.
    return "authenticationerror" in message.lower()


def _hint(status: int, message: str) -> str:
    if _is_auth_error(message):
        return " (check the provider key set in the gateway, e.g. OPENCODE_API_KEY)"
    if status in (401, 403) or "no connected db" in message.lower():
        # Without a database, LiteLLM answers an unknown key with "No connected db".
        return " (check LLM_API_KEY: the gateway does not accept it)"
    return ""


def _retry_after(value: str | None) -> float | None:
    try:
        return max(0.0, float(value)) if value else None
    except ValueError:
        return None  # an HTTP date: use the normal backoff


def _reason(e: Exception) -> str:
    reason = getattr(e, "reason", None) or e
    return str(reason) or reason.__class__.__name__
