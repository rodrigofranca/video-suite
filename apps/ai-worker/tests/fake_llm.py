"""A fake OpenAI-compatible chat completions server, for tests.

Tests queue answers with `answer()` and read what the client sent in
`requests`. Run as a script, it answers every prompt with a deterministic
answer built from the transcript lines it receives (a summary, clips, or
lines marked with the language they were to be translated into), which is
enough to try the whole pipeline without a real model:

    python tests/fake_llm.py --port 8000
"""

from __future__ import annotations

import argparse
import json
import re
import threading
from collections import deque
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class FakeLLM:
    def __init__(self, port: int = 0, model: str = "fake-model"):
        self.model = model
        self.requests: list[dict] = []  # {"path", "headers", "body"}
        self._answers: deque = deque()
        self.default = None  # used when no answer is queued: fn(body) -> (status, payload, headers)
        fake = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):  # keep test output quiet
                pass

            def _reply(self, status, payload, headers=None):
                raw = payload if isinstance(payload, bytes) else json.dumps(payload).encode()
                self.send_response(status)
                self.send_header("Content-Type", "application/json")
                for k, v in (headers or {}).items():
                    self.send_header(k, v)
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)

            def do_GET(self):
                fake.requests.append({"path": self.path, "headers": dict(self.headers), "body": None})
                if self.path.rstrip("/").endswith("/models"):
                    self._reply(200, {"object": "list", "data": [{"id": fake.model, "object": "model"}]})
                else:
                    self._reply(404, {"error": {"message": "not found"}})

            def do_POST(self):
                length = int(self.headers.get("Content-Length") or 0)
                body = json.loads(self.rfile.read(length) or b"{}")
                fake.requests.append({"path": self.path, "headers": dict(self.headers), "body": body})
                if fake._answers:
                    answer = fake._answers.popleft()
                elif fake.default is not None:
                    answer = fake.default(body)
                else:
                    answer = (500, {"error": {"message": "no answer queued"}}, None)
                self._reply(*answer)

        self.server = ThreadingHTTPServer(("127.0.0.1", port), Handler)
        self.base_url = f"http://127.0.0.1:{self.server.server_address[1]}/v1"
        self._thread = threading.Thread(target=self.server.serve_forever, daemon=True)

    def __enter__(self) -> "FakeLLM":
        self._thread.start()
        return self

    def __exit__(self, *exc) -> None:
        self.server.shutdown()
        self.server.server_close()

    def answer(
        self, content: str | dict, *, finish_reason: str = "stop", model: str | None = None, litellm_model: str = ""
    ) -> None:
        """Queue a successful completion. A dict is sent as its JSON text.
        litellm_model adds the header LiteLLM uses to name the real model."""
        text = json.dumps(content, ensure_ascii=False) if isinstance(content, dict) else content
        headers = {"x-litellm-model-name": litellm_model} if litellm_model else None
        self._answers.append((200, completion(text, model or self.model, finish_reason), headers))

    def fail(self, status: int, message: str, headers: dict | None = None) -> None:
        """Queue an error response."""
        self._answers.append((status, {"error": {"message": message, "code": status}}, headers))


def completion(text: str, model: str, finish_reason: str = "stop") -> dict:
    return {
        "id": "chatcmpl-fake",
        "object": "chat.completion",
        "model": model,
        "choices": [{"index": 0, "message": {"role": "assistant", "content": text}, "finish_reason": finish_reason}],
        "usage": {"prompt_tokens": 100, "completion_tokens": 50, "total_tokens": 150},
    }


_LINE = re.compile(r"^\[(\d+(?::\d\d){1,2})\] (.+)$", re.MULTILINE)


_SEGMENT = re.compile(r"^\[(\d+(?:\.\d+)?)-(\d+(?:\.\d+)?)\] (.+)$", re.MULTILINE)


def demo_clips(prompt: str) -> dict:
    """Clips of about 20 to 40 seconds made of consecutive segments."""
    clips, start, text = [], None, []
    for s, e, t in _SEGMENT.findall(prompt):
        s, e = float(s), float(e)
        start = s if start is None else start
        text.append(t)
        if e - start >= 20:
            clips.append({"start": start, "end": e, "title": " ".join(" ".join(text).split()[:5]).rstrip(".,"),
                          "reason": "(Simulated model) A self-contained moment.", "score": 9 - len(clips)})
            start, text = None, []
        if len(clips) == 3:
            break
    return {"clips": clips}


_TARGET = re.compile(r"^Translate these subtitle lines of a video from .+? into (.+)\.$", re.MULTILINE)
_NUMBERED = re.compile(r"^\d+\. (.+)$", re.MULTILINE)


def demo_translation(prompt: str) -> dict:
    """Each line marked with the target language, e.g. "(English) Olá"."""
    target = _TARGET.search(prompt).group(1)
    lines = _NUMBERED.findall(prompt.split("\nLines:\n", 1)[1])
    return {"lines": [f"({target}) {line}" for line in lines]}


def demo_answer(body: dict) -> tuple[int, dict, None]:
    """A plausible answer built from the prompt's transcript lines: clips,
    a translation or an analysis."""
    prompt = next((m["content"] for m in body.get("messages", []) if m.get("role") == "user"), "")
    if "standalone clips" in prompt:
        return 200, completion(json.dumps(demo_clips(prompt)), "fake-model"), None
    if prompt.startswith("Translate these subtitle lines"):
        return 200, completion(json.dumps(demo_translation(prompt), ensure_ascii=False), "fake-model"), None
    lines = _LINE.findall(prompt)
    text = " ".join(t for _, t in lines)
    sentences = re.split(r"(?<=[.!?])\s+", text)
    words = [w.strip(".,!?").lower() for w in text.split() if len(w.strip(".,!?")) > 5]
    tags = list(dict.fromkeys(words))[:5] or ["video"]
    chapters = [{"start": ts, "title": " ".join(t.split()[:4]).rstrip(".,")} for ts, t in lines[:: max(1, len(lines) // 3)]]
    answer = {
        "summary": "(Simulated model) " + " ".join(sentences[:2]),
        "tags": tags,
        "chapters": chapters if len(chapters) >= 2 else [],
    }
    return 200, completion(json.dumps(answer, ensure_ascii=False), "fake-model"), None


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--host", default="0.0.0.0")
    parser.add_argument("--port", type=int, default=8000)
    args = parser.parse_args()
    fake = FakeLLM(args.port)
    fake.server.server_close()  # rebind on the requested host
    fake.server = ThreadingHTTPServer((args.host, args.port), fake.server.RequestHandlerClass)
    fake.default = demo_answer
    print(f"fake LLM listening on {args.host}:{args.port}", flush=True)
    fake.server.serve_forever()


if __name__ == "__main__":
    main()
