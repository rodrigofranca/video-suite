# AI worker image (Python, faster-whisper). Build context: the repository root.
# The Python wheels bundle what they need (CTranslate2, FFmpeg libraries for
# audio decoding), so no system packages are installed.

FROM python:3.13.17-slim-bookworm
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy
RUN pip install --no-cache-dir uv==0.13.5

WORKDIR /app
# Dependencies first, so they are cached between code changes. The uv download
# cache lives in a build cache mount, outside the image.
COPY apps/ai-worker/pyproject.toml apps/ai-worker/uv.lock ./
RUN --mount=type=cache,target=/root/.cache/uv uv sync --frozen --no-dev --no-install-project
COPY apps/ai-worker/src ./src
RUN --mount=type=cache,target=/root/.cache/uv uv sync --frozen --no-dev

# Whisper models download here on first start; mount a volume to keep them.
RUN useradd --create-home app && mkdir /models && chown app /models
ENV HF_HOME=/models \
    PATH="/app/.venv/bin:$PATH"
USER app
ENTRYPOINT ["python", "-m", "ultramedia_ai"]
