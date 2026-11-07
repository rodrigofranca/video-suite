"""Entry point: python -m ultramedia_ai"""

from __future__ import annotations

import logging
import signal
import sys
import threading

import redis

from . import contracts
from .analyze import Analyzer
from .clips import ClipSuggester
from .config import ConfigError, from_env
from .llm import ChatClient
from .queue import Consumer
from .runner import Runner
from .storage import Storage
from .store import Store
from .transcribe import Transcriber


def main() -> int:
    logging.basicConfig(
        level=logging.INFO,
        stream=sys.stdout,
        format="time=%(asctime)s level=%(levelname)s msg=%(message)s",
    )
    for noisy in ("botocore", "boto3", "urllib3", "faster_whisper", "httpx", "httpx2", "huggingface_hub"):
        logging.getLogger(noisy).setLevel(logging.WARNING)
    log = logging.getLogger("ai-worker")

    try:
        cfg = from_env()
    except ConfigError as e:
        log.error("ai worker stopped: %s", e)
        return 1

    stop = threading.Event()

    def on_signal(signum, _frame):
        log.info("received %s, stopping", signal.Signals(signum).name)
        stop.set()

    signal.signal(signal.SIGINT, on_signal)
    signal.signal(signal.SIGTERM, on_signal)

    try:
        client = redis.Redis.from_url(cfg.redis_url, decode_responses=True, socket_connect_timeout=5)
        client.ping()
        store = Store.connect(cfg.database_url)
        storage = Storage(cfg)
        storage.check()
    except Exception as e:  # noqa: BLE001 - report any startup failure the same way
        log.error("ai worker stopped: could not connect: %s", e)
        return 1
    log.info("connected redis=%s storage=%s", cfg.redis_url.split("@")[-1], cfg.s3_endpoint)

    transcriber = Transcriber(cfg, storage, store, log)
    try:
        transcriber.load_model()  # fail at startup, not on the first job
    except Exception as e:  # noqa: BLE001
        log.error("ai worker stopped: could not load the Whisper model: %s", e)
        return 1

    llm = None
    if cfg.llm_base_url:
        llm = ChatClient(cfg.llm_base_url, cfg.llm_api_key, cfg.llm_model, timeout=cfg.llm_timeout_seconds)
        check_gateway(llm, log)
    else:
        log.warning("no language model configured (LLM_BASE_URL): analysis and clip jobs will fail")

    def enqueue(msg: contracts.Message) -> None:
        client.xadd(msg.stream, msg.values())

    handlers = {
        contracts.JOB_TRANSCRIBE_AUDIO: transcriber,
        contracts.JOB_ANALYZE_TRANSCRIPT: Analyzer(cfg, store, llm, log),
        contracts.JOB_SUGGEST_CLIPS: ClipSuggester(cfg, store, llm, log),
    }
    runner = Runner(store, handlers, enqueue, log, stop, cfg.job_timeout_seconds)
    consumer = Consumer(client, runner.process, log, cfg.worker_name, stop)
    log.info(
        "consuming jobs stream=%s group=%s consumer=%s",
        consumer.stream, consumer.group, cfg.worker_name,
    )
    consumer.run()
    store.close()
    log.info("ai worker shut down")
    return 0


def check_gateway(llm: ChatClient, log: logging.Logger) -> None:
    """Warn early about a gateway that is down or lacks the model. Not fatal:
    transcription works without it, and the gateway may start later."""
    try:
        models = llm.models()
    except Exception as e:  # noqa: BLE001
        log.warning("language model gateway at %s not reachable yet: %s", llm.host, e)
        return
    if llm.model not in models:
        log.warning("language model gateway at %s has no model %r (it has: %s)", llm.host, llm.model, ", ".join(models) or "none")
        return
    log.info("language model: %s via gateway %s", llm.model, llm.host)


if __name__ == "__main__":
    sys.exit(main())
