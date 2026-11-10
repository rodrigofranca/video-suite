"""The gateway client against a fake OpenAI-compatible server."""

import pytest

from fake_llm import FakeLLM
from ultramedia_ai.llm import ChatClient, LLMError

MESSAGES = [{"role": "user", "content": "hi"}]


@pytest.fixture
def fake():
    with FakeLLM() as f:
        yield f


def client(fake, **kw) -> ChatClient:
    return ChatClient(fake.base_url, "sk-test-key-123456", "ultramedia-text", timeout=5, **kw)


def test_completion(fake):
    fake.answer("hello", model="claude-haiku-5-5")
    c = client(fake).complete(MESSAGES, max_tokens=50)
    assert (c.text, c.model, c.finish_reason, c.prompt_tokens, c.completion_tokens) == (
        "hello", "claude-haiku-5-5", "stop", 100, 50
    )
    req = fake.requests[0]
    assert req["path"] == "/v1/chat/completions"
    assert req["headers"]["Authorization"] == "Bearer sk-test-key-123456"
    assert req["body"] == {"model": "ultramedia-text", "messages": MESSAGES, "max_tokens": 50}


@pytest.mark.parametrize(
    "header,want",
    [("3.5e-05", 3.5e-05), ("0.0012", 0.0012), ("", None), ("0.0", None), ("-1", None), ("abc", None)],
)
def test_reads_the_cost_litellm_reports(fake, header, want):
    """No header, a bad one, or zero for a call that used tokens: the price is unknown."""
    fake.answer("hi", cost=header)
    cost = client(fake).complete(MESSAGES, max_tokens=5).cost_usd
    assert cost == pytest.approx(want) if want is not None else cost is None


def test_reports_the_model_behind_a_litellm_alias(fake):
    fake.answer("hi", model="ultramedia-text", litellm_model="openai/claude-haiku-5-5")
    assert client(fake).complete(MESSAGES, max_tokens=5).model == "claude-haiku-5-5"


def test_retries_transient_errors(fake):
    fake.fail(503, "upstream overloaded")
    fake.fail(429, "slow down", headers={"Retry-After": "7"})
    fake.answer("finally")
    waits = []
    assert client(fake, backoff=1).complete(MESSAGES, max_tokens=5, sleep=waits.append).text == "finally"
    assert waits == [1, 7]  # backoff, then the longer Retry-After
    assert len(fake.requests) == 3


def test_gives_up_after_the_last_attempt(fake):
    for _ in range(3):
        fake.fail(502, "bad gateway")
    with pytest.raises(LLMError, match="answered 502: bad gateway"):
        client(fake).complete(MESSAGES, max_tokens=5, sleep=lambda s: None)
    assert len(fake.requests) == 3


def test_does_not_retry_client_errors_and_hides_keys(fake):
    fake.fail(401, "Incorrect API key provided: sk-live-abcdef1234567890. Check your key.")
    with pytest.raises(LLMError) as e:
        client(fake).complete(MESSAGES, max_tokens=5, sleep=lambda s: None)
    assert len(fake.requests) == 1
    assert "answered 401" in str(e.value)
    assert "sk-live" not in str(e.value) and "[redacted]" in str(e.value)


def test_auth_errors_are_not_retried_and_get_a_hint(fake):
    fake.fail(500, "litellm.AuthenticationError: AuthenticationError: OpenAIException - The api_key client option must be set")
    with pytest.raises(LLMError, match=r"check the provider key set in the gateway"):
        client(fake).complete(MESSAGES, max_tokens=5, sleep=lambda s: pytest.fail("should not retry"))
    fake.fail(400, "No connected db.")  # LiteLLM without a database, given an unknown key
    with pytest.raises(LLMError, match=r"check LLM_API_KEY"):
        client(fake).complete(MESSAGES, max_tokens=5)


def test_no_credits_is_not_retried(fake):
    fake.fail(402, "Insufficient account funds")
    with pytest.raises(LLMError, match=r"answered 402: Insufficient account funds \(the provider account has no credits left\)"):
        client(fake).complete(MESSAGES, max_tokens=5, sleep=lambda s: pytest.fail("should not retry"))


def test_long_retry_after_fails_fast(fake):
    fake.fail(429, "quota", headers={"Retry-After": "3600"})
    with pytest.raises(LLMError, match="429"):
        client(fake).complete(MESSAGES, max_tokens=5, sleep=lambda s: pytest.fail("should not wait"))


def test_unreachable_gateway():
    c = ChatClient("http://127.0.0.1:9/v1", "", "m", timeout=2, attempts=2)
    with pytest.raises(LLMError, match="could not reach the language model gateway at 127.0.0.1:9"):
        c.complete(MESSAGES, max_tokens=5, sleep=lambda s: None)


def test_check_stops_retries(fake):
    fake.fail(503, "down")

    class Stop(Exception):
        pass

    calls = []

    def check():
        calls.append(1)
        if len(calls) > 1:
            raise Stop()

    with pytest.raises(Stop):
        client(fake).complete(MESSAGES, max_tokens=5, sleep=lambda s: None, check=check)
    assert len(fake.requests) == 1


def test_models(fake):
    assert client(fake).models() == ["fake-model"]
    assert client(fake).host.startswith("127.0.0.1:")
