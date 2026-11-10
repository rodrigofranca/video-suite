"""The jobs that ask the language model share one shape (docs/adr/0036), a
Template Method: LLMTask.__call__ runs the steps every such job takes, in
order, and each job fills in its own.

1. read the input and check it (``check_input``);
2. make sure a model is configured;
3. plan the requests from what is stored (``plan``), or decide there is
   nothing to ask;
4. check every prompt's length before spending a token;
5. ask, checking each answer and asking once more when it cannot be used
   (answers.ask), with progress when there are several requests;
6. save the result (``save``) and log it with the model and the tokens.

A new job that asks the model is a subclass with an input, a system prompt,
``plan`` and ``save``.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from typing import Any, Callable

from . import contracts
from .answers import MAX_TOKENS, NotConfigured, ask, count
from .config import Config
from .llm import ChatClient, Completion
from .runner import JobContext
from .store import Store


@dataclass
class Request:
    """One request to the model: the prompt, and how to read the answer
    (raising answers.InvalidAnswer when it cannot be used)."""

    prompt: str
    parse: Callable[[str], Any]
    max_tokens: int = MAX_TOKENS


@dataclass
class Plan:
    """What a job will ask: its requests, a few words on their size for the
    log, and whatever the job needs again when it saves."""

    requests: list[Request]
    detail: str
    state: dict = field(default_factory=dict)


@dataclass
class Usage:
    """What the requests used together."""

    model: str = ""
    prompt_tokens: int = 0
    completion_tokens: int = 0
    unknown: bool = False  # the gateway did not report the tokens of some answer

    def add(self, completion: Completion) -> None:
        self.model = completion.model or self.model
        if completion.prompt_tokens is None or completion.completion_tokens is None:
            self.unknown = True
        self.prompt_tokens += completion.prompt_tokens or 0
        self.completion_tokens += completion.completion_tokens or 0

    def __str__(self) -> str:
        tokens_in = None if self.unknown and not self.prompt_tokens else self.prompt_tokens
        tokens_out = None if self.unknown and not self.completion_tokens else self.completion_tokens
        return f"model {self.model}, {count(tokens_in)} tokens in, {count(tokens_out)} out"


class LLMTask:
    """A job that asks the language model. Subclasses set ``input_type``,
    ``system_prompt`` and ``what``, and write ``plan`` and ``save``."""

    input_type: type  # the job's input contract, with from_json
    system_prompt: str
    what: str  # what the job asks for, for the log: "a summary, tags and chapters"

    def __init__(self, cfg: Config, store: Store, client: ChatClient | None, log: logging.Logger):
        self.cfg, self.store, self.client, self.log = cfg, store, client, log

    def __call__(self, ctx: JobContext, payload: str) -> None:
        inp = self.input_type.from_json(payload)
        self.check_input(inp)
        if self.client is None:
            raise NotConfigured("no language model is configured: set LLM_BASE_URL (see .env.example)")
        plan = self.plan(ctx, inp)
        if plan is None:
            return
        for req in plan.requests:
            check_length(req.prompt, self.cfg)

        ctx.log.info("asking model %s at %s for %s (%s)", self.client.model, self.client.host, self.what, plan.detail)
        answers, usage = [], Usage()
        for i, req in enumerate(plan.requests):
            ctx.check()
            messages = [{"role": "system", "content": self.system_prompt}, {"role": "user", "content": req.prompt}]
            answer, completion = ask(self.client, ctx, messages, req.parse, req.max_tokens)
            answers.append(answer)
            usage.add(completion)
            if len(plan.requests) > 1:
                ctx.progress(min(95, (i + 1) * 95 // len(plan.requests)))
        result = self.save(ctx, inp, plan, answers, usage)
        ctx.log.info("%s; %s", result, usage)

    # ---- The steps each job fills in ----

    def check_input(self, inp) -> None:
        """Rejects an input the job cannot use. Every input names a video."""
        if not contracts.is_id(inp.asset_id):
            raise ValueError(f"invalid input: asset_id {inp.asset_id!r}")

    def plan(self, ctx: JobContext, inp) -> Plan | None:
        """The requests to send, or None when there is nothing to ask (after
        saying why in the log, and saving what that means)."""
        raise NotImplementedError

    def save(self, ctx: JobContext, inp, plan: Plan, answers: list, usage: Usage) -> str:
        """Saves the answers and returns what was saved, for the log."""
        raise NotImplementedError


def check_length(prompt: str, cfg: Config) -> None:
    if len(prompt) > cfg.llm_max_input_chars:
        raise ValueError(
            f"the transcript is too long for one request ({len(prompt)} characters, "
            f"limit {cfg.llm_max_input_chars}, set by LLM_MAX_INPUT_CHARS)"
        )
